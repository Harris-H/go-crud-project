package main

// 导入模块
import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"log"
	"net/http"
	"strconv"
)

var db *sql.DB
var cnt int

// Person 自定义Person类
type Person struct {
	Id        int    `json:"id"`
	FirstName string `json:"first_name" form:"first_name"`
	LastName  string `json:"last_name" form:"last_name"`
}

func (g *Person) MarshalBinary() (data []byte, err error) {
	return json.Marshal(g)
}

func (g *Person) UnmarshalBinary(data []byte) (err error) {
	return json.Unmarshal(data, g)
}

func (p *Person) get(c *redis.Client, ctx *context.Context) (person Person, err error) {
	err = c.Get(*ctx, strconv.Itoa(p.Id)).Scan(p)
	if err != nil {
		log.Fatalln("get fail!")
		return
	}
	person = *p
	return
}

func (p *Person) getAll(c *redis.Client, ctx *context.Context) (persons []Person, err error) {
	var ids []string
	ids, err = c.Keys(*ctx, "[1-9]*").Result()
	for _, id := range ids {
		myid, _ := strconv.Atoi(id)
		var person = Person{
			Id: myid,
		}
		person, _ = person.get(c, ctx)
		persons = append(persons, person)
	}
	return
}

func (p *Person) add(c *redis.Client, ctx *context.Context) (Id int, err error) {
	// HMSet 批量设置 map[string]interface{}{"age": 18, "sex": "male"}
	cnt += 1
	Id = cnt
	p.Id = Id
	_, err = c.Set(*ctx, strconv.Itoa(cnt), p, 0).Result()
	if err != nil {
		return
	}
	return
}

func (p *Person) update(c *redis.Client, ctx *context.Context) (rows int, err error) {
	if c.Exists(*ctx, strconv.Itoa(p.Id)).Val() == 1 {
		_, err = c.Set(*ctx, strconv.Itoa(p.Id), p, 0).Result()
		if err != nil {
			return
		}
		rows = 1
	}
	return
}

func (p *Person) del(c *redis.Client, ctx *context.Context) (rows int, err error) {
	if c.Exists(*ctx, strconv.Itoa(p.Id)).Val() == 1 {
		var my int64
		my, err = c.Del(*ctx, strconv.Itoa(p.Id)).Result()
		rows = int(my)
		if err != nil {
			return
		}
	}
	return
}

func test(db *sql.DB) {
	rows, err := db.Query("select * from person")
	if err != nil {
		return
	}
	defer rows.Close()
	var persons []Person
	for rows.Next() {
		var person Person
		rows.Scan(&person.Id, &person.FirstName, &person.LastName)
		persons = append(persons, person)
	}
	fmt.Println(persons)
}
func main() {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	rdb.FlushDB(ctx)
	defer rdb.Close()
	_, err := rdb.Ping(ctx).Result() // PING, <nil>
	if err != nil {
		fmt.Println("connect redis failed:", err)
		return
	}
	//创建路由引擎
	router := gin.Default()
	//查询,返回所有对象和对象个数
	router.GET("/persons", func(context *gin.Context) {
		p := Person{}
		persons, err := p.getAll(rdb, &ctx)
		if err != nil {
			log.Fatalln(err)
		}
		context.JSON(http.StatusOK, gin.H{
			"result": persons,
			"count":  len(persons),
		})
	})

	//根据id查询
	router.GET("/person/:id", func(context *gin.Context) {
		var result gin.H
		id := context.Param("id")

		Id, err := strconv.Atoi(id)
		if err != nil {
			log.Fatalln(err)
		}
		p := Person{
			Id: Id,
		}
		person, err := p.get(rdb, &ctx)
		if err != nil {
			result = gin.H{
				"result": nil,
				"count":  0,
			}
		} else {
			result = gin.H{
				"result": person,
				"count":  1,
			}
		}
		context.JSON(http.StatusOK, result)
	})

	//创建person
	router.POST("/person", func(context *gin.Context) {
		var p Person
		err := context.Bind(&p)
		if err != nil {
			log.Fatalln(err)
		}

		Id, err := p.add(rdb, &ctx)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(Id)
		name := p.FirstName + " " + p.LastName
		context.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf(" %s 成功创建", name),
		})
	})

	//更新update
	router.PUT("/person/:id", func(context *gin.Context) {
		var (
			p      Person
			buffer bytes.Buffer
		)

		id := context.Param("id")
		Id, err := strconv.Atoi(id)
		if err != nil {
			log.Fatalln(err)
		}
		err = context.Bind(&p)
		if err != nil {
			log.Fatalln(err)
		}
		p.Id = Id
		rows, err := p.update(rdb, &ctx)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(rows)
		buffer.WriteString(p.FirstName)
		buffer.WriteString(" ")
		buffer.WriteString(p.LastName)
		name := buffer.String()

		context.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("成功更新到%s", name),
		})
	})

	//删除person
	router.DELETE("/person/:id", func(context *gin.Context) {
		id := context.Param("id")

		Id, err := strconv.ParseInt(id, 10, 10)
		if err != nil {
			log.Fatalln(err)
		}
		p := Person{Id: int(Id)}
		rows, err := p.del(rdb, &ctx)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("delete rows: ", rows)

		context.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("成功删除用户：%s", id),
		})
	})
	router.Run(":8080")
}
