package main

import (
	"github.com/ygb616/web"
	"github.com/ygb616/web/rpc"
	"log"
)

func main() {
	engine := web.Default()
	client := rpc.NewHttpClient()
	group := engine.Group("order")
	group.Get("find", func(ctx *web.Context) {
		params := make(map[string]any)
		params["id"] = 1000
		params["name"] = "张三"
		body, err := client.Get("http://localhost:9002/order/find", params)
		if err != nil {
			panic(err)
		}
		log.Println(string(body))
	})

	group.Post("find", func(ctx *web.Context) {
		params := make(map[string]any)
		params["id"] = 1000
		params["name"] = "张三"
		body, err := client.PostJson("http://localhost:9002/order/find", params)
		if err != nil {
			panic(err)
		}
		log.Println(string(body))
	})

	engine.Run(9003)
}
