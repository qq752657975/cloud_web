package main

import (
	"github.com/ygb616/goodscenter/api"
	"github.com/ygb616/goodscenter/model"
	"github.com/ygb616/web"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
)

func main() {
	engine := web.Default()

	group := engine.Group("goods")
	group.Get("find", func(ctx *web.Context) {
		goods := &model.Goods{Id: 1000, Name: "9002的商品"}
		ctx.JSON(http.StatusOK, &model.Result{Code: 200, Msg: "success", Data: goods})
	})

	listen, _ := net.Listen("tcp", ":9111")
	server := grpc.NewServer()
	api.RegisterGoodsApiServer(server, &api.GoodsRpcService{})
	err := server.Serve(listen)
	log.Println(err)
	engine.Run(9002)
}
