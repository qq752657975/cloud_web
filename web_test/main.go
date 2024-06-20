package main

import (
	"errors"
	"fmt"
	"github.com/ygb616/web"
	myLog "github.com/ygb616/web/log"
	"log"
	"net/http"
)

type BlogResponse struct {
	Success bool
	Code    int
	Data    any
	Msg     string
}
type BlogNoDataResponse struct {
	Success bool
	Code    int
	Msg     string
}

func (b *BlogResponse) Error() string {
	return b.Msg
}

func (b *BlogResponse) Response(success bool, code int, msg string) any {
	if b.Data == nil {
		return &BlogNoDataResponse{
			Success: success,
			Code:    code,
			Msg:     msg,
		}
	}
	return b
}

func login() *BlogResponse {
	return &BlogResponse{
		Success: false,
		Code:    -999,
		Data:    nil,
		Msg:     "账号密码错误",
	}
}

type User struct {
	Name      string   `xml:"name" json:"name" web:"required"`
	Age       int      `xml:"age" json:"age" validate:"required,max=50,min=18"`
	Addresses []string `json:"addresses"`
}

func Log(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) {
		fmt.Println("打印请求参数")
		next(ctx)
		fmt.Println("返回执行时间")
	}
}

func main() {
	engine := web.Default()
	engine.RegisterErrorHandler(func(err error) (int, any) {
		var e *BlogResponse
		switch {
		case errors.As(err, &e):
			return http.StatusOK, e.Response(false, e.Code, e.Msg)
		default:
			return http.StatusInternalServerError, "Internal Server Error"
		}
	})
	g := engine.Group("user")
	engine.Logger.Level = myLog.LevelError
	engine.Logger.Formatter = &myLog.TextFormatter{}
	//engine.Logger.SetLogPath("./log")
	g.Get("/get", func(ctx *web.Context) {
		err := login()
		ctx.HandlerWithError(http.StatusOK, &User{}, err)

	})
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

	g.Post("/formPost", func(ctx *web.Context) {
		data, _ := ctx.GetPostForm("name")
		ctx.JSON(http.StatusOK, data)
	})

	g.Post("/jsonParam", func(ctx *web.Context) {
		user := &User{}
		err := ctx.BindJson(user)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})

	g.Post("/xmlParam", func(ctx *web.Context) {
		a := 1
		b := 0
		c := a / b
		fmt.Sprintf(string(c))
		user := &User{}
		//user := User{}
		err := ctx.BindXML(user)

		if err == nil {
			err := ctx.JSON(http.StatusOK, user)
			if err != nil {
				return
			}
		} else {
			log.Println(err)
		}
	})

	engine.Run(8111)
}
