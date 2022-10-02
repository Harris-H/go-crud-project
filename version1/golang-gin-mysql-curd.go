package main

// 导入模块
import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"strconv"
)

var db *sql.DB

// Person 自定义Person类
type Person struct {
	Id        int    `json:"id"`
	FirstName string `json:"first_name" form:"first_name"`
	LastName  string `json:"last_name" form:"last_name"`
}

func (p *Person) get(db *sql.DB) (person Person, err error) {
	row := db.QueryRow("SELECT id,first_name,last_name from person where id=?", p.Id)
	err = row.Scan(&person.Id, &person.FirstName, &person.LastName)
	if err != nil {
		return
	}
	return
}

func (p *Person) getAll(db *sql.DB) (persons []Person, err error) {
	rows, err := db.Query("select id,first_name,last_name from person")
	fmt.Println(rows)
	if err != nil {
		return
	}
	for rows.Next() {
		var person Person
		rows.Scan(&person.Id, &person.FirstName, &person.LastName)
		persons = append(persons, person)
	}
	defer rows.Close()
	return
}

func (p *Person) add(db *sql.DB) (Id int, err error) {
	stmt, err := db.Prepare("INSERT into person(first_name,last_name) values (?,?)")
	if err != nil {
		return
	}
	rs, err := stmt.Exec(p.FirstName, p.LastName)
	if err != nil {
		return
	}
	id, err := rs.LastInsertId()
	if err != nil {
		log.Fatalln(err)
	}
	Id = int(id)
	defer stmt.Close()
	return
}

func (p *Person) update(db *sql.DB) (rows int, err error) {
	stmt, err := db.Prepare("update person set first_name=?,last_name=? where id=?")
	if err != nil {
		log.Fatalln(err)
	}
	rs, err := stmt.Exec(p.FirstName, p.LastName, p.Id)
	if err != nil {
		log.Fatalln(err)
	}
	row, err := rs.RowsAffected()
	if err != nil {
		log.Fatalln(err)
	}
	rows = int(row)
	defer stmt.Close()
	return
}

func (p *Person) del(db *sql.DB) (rows int, err error) {
	stmt, err := db.Prepare("delete from person where id=?")
	if err != nil {
		log.Fatalln(err)
	}
	rs, err := stmt.Exec(p.Id)
	if err != nil {
		log.Fatalln(err)
	}
	row, err := rs.RowsAffected()
	if err != nil {
		log.Fatalln(err)
	}
	rows = int(row)
	defer stmt.Close()
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
	var err error
	db, err := sql.Open("mysql", "root:Hh013227369688@tcp(127.0.0.1:3306)/go_project01?parseTime=true")
	if err != nil {
		log.Fatal(err.Error())
	}
	//defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatal(err.Error())
	}
	//test(db)
	//创建路由引擎
	router := gin.Default()

	//查询,返回所有对象和对象个数
	router.GET("/persons", func(context *gin.Context) {
		p := Person{}
		persons, err := p.getAll(db)
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
		person, err := p.get(db)
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

		Id, err := p.add(db)
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
		rows, err := p.update(db)
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
		rows, err := p.del(db)
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
