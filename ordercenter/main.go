package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ygb616/goodscenter/api"
	"github.com/ygb616/goodscenter/model"
	"github.com/ygb616/goodscenter/service"
	"github.com/ygb616/web"
	"github.com/ygb616/web/rpc"
	"net/http"
)

func main() {
	engine := web.Default()
	client := rpc.NewHttpClient()
	g := engine.Group("order")
	client.RegisterHttpService("goodsService", &service.GoodsService{})
	g.Get("/find", func(ctx *web.Context) {
		//查询商品
		session := client.Session()
		v := &model.Result{}
		bytes, err := session.Do("goods", "Find").(*service.GoodsService).Find(nil)
		if err != nil {
			ctx.Logger.Error(err)
		}
		fmt.Println(string(bytes))
		json.Unmarshal(bytes, v)
		ctx.JSON(http.StatusOK, v)
	})

	//g.Get("/findGrpc", func(ctx *web.Context) {
	//	//查询商品
	//	var serviceHost = "127.0.0.1:9111"
	//	conn, err := grpc.Dial(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//	defer conn.Close()
	//
	//	client := api.NewGoodsApiClient(conn)
	//	rsp, err := client.Find(context.TODO(), &api.GoodsRequest{})
	//
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//	ctx.JSON(http.StatusOK, rsp)
	//})
	g.Get("/findGrpc", func(ctx *web.Context) {
		config := rpc.DefaultGrpcClientConfig()
		config.Address = "localhost:9111"
		client, _ := rpc.NewGrpcClient(config)
		defer client.Conn.Close()
		goodsApiClient := api.NewGoodsApiClient(client.Conn)
		goodsResponse, _ := goodsApiClient.Find(context.Background(), &api.GoodsRequest{})
		ctx.JSON(http.StatusOK, goodsResponse)
	})

	engine.Run(9003)
}
