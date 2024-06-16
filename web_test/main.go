package main

import (
	"fmt"
	"github.com/ygb616/web"
	"log"
	"net/http"
)

type User struct {
	Name string `xml:"name"`
}

func Log(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) {
		fmt.Println("打印请求参数")
		next(ctx)
		fmt.Println("返回执行时间")
	}
}

func main() {
	engine := web.New()
	g := engine.Group("user")
	g.Use(func(nextHandler web.HandlerFunc) web.HandlerFunc {
		return func(ctx *web.Context) {
			fmt.Println("pre handler")
			nextHandler(ctx)
			fmt.Println("post handler")
		}
	})

	g.Get("/get", func(ctx *web.Context) {
		fmt.Println("测试get方法")
		fmt.Fprintf(ctx.W, "测试get方法")
	}, Log)
	g.Get("/hello", func(ctx *web.Context) {
		err := ctx.HTML(http.StatusOK, "<h1>你好 go微服务框架</h1>")
		if err != nil {
			panic(err)
		}
	})

	g.Get("/htmlTemplate", func(ctx *web.Context) {
		user := User{
			Name: "ygb616",
		}
		err := ctx.HTMLTemplate("login.html", user, "tpl/login.html", "tpl/header.html")
		if err != nil {
			log.Println(err)
		}
	})

	g.Get("/htmlTemplateGlob", func(ctx *web.Context) {
		user := User{
			Name: "ygb616",
		}
		err := ctx.HTMLTemplateGlob("login.html", user, "tpl/*.html")
		if err != nil {
			log.Println(err)
		}
	})
	engine.LoadTemplate("tpl/*.html")
	g.Get("/template", func(ctx *web.Context) {
		user := User{
			Name: "ygb616",
		}
		err := ctx.Template("login.html", user)
		if err != nil {
			log.Println(err)
		}
	})

	g.Get("/json", func(ctx *web.Context) {
		user := User{
			Name: "ygb616",
		}
		err := ctx.JSON(http.StatusOK, user)
		if err != nil {
			log.Println(err)
		}
	})

	g.Get("/xml", func(ctx *web.Context) {
		user := User{
			Name: "ygb616",
		}
		err := ctx.XML(http.StatusOK, user)
		if err != nil {
			log.Println(err)
		}
	})

	g.Get("/excel", func(ctx *web.Context) {
		ctx.File("tpl/test.xlsx")
	})

	g.Get("/excelName", func(ctx *web.Context) {
		ctx.FileAttachment("tpl/test.xlsx", "aaaa.xlsx")
	})

	g.Get("/fs", func(ctx *web.Context) {
		ctx.FileFromFS("test.xlsx", http.Dir("tpl"))
	})

	g.Get("/redirect", func(ctx *web.Context) {
		ctx.Redirect(http.StatusFound, "/user/template")
	})

	g.Get("/string", func(ctx *web.Context) {
		ctx.String(http.StatusOK, "%s 是由 %s 制作 \n", "goweb框架", "go微服务框架")

	})

	engine.Run(8111)
}
