package main

import (
	"github.com/ygb616/goodscenter/model"
	"github.com/ygb616/web"
	"net/http"
)

func main() {
	engine := web.Default()

	group := engine.Group("goods")
	group.Get("find", func(ctx *web.Context) {
		goods := &model.Goods{Id: 1000, Name: "9002的商品"}
		ctx.JSON(http.StatusOK, &model.Result{Code: 200, Msg: "success", Data: goods})
	})

	engine.Run(9002)
}
