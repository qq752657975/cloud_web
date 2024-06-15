package main

import (
	"fmt"
	"github.com/ygb616/web"
)

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
	g.Get("/get/*/hello", func(ctx *web.Context) {
		fmt.Fprintf(ctx.W, "测试中间*方法")
	})

	engine.Run(8111)
}
