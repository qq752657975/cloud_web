package web

import (
	"context"
	"golang.org/x/time/rate"
	"net/http"
	"time"
)

// Limiter 返回一个限流中间件
func Limiter(limit, cap int) MiddlewareFunc {
	li := rate.NewLimiter(rate.Limit(limit), cap) // 创建限流器
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			// 实现限流
			con, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Second) // 设置超时上下文
			defer cancel()                                                                         // 确保上下文取消
			err := li.WaitN(con, 1)                                                                // 请求令牌
			if err != nil {
				ctx.String(http.StatusForbidden, "限流了") // 如果限流，返回403状态码
				return
			}
			next(ctx) // 调用下一个处理函数
		}
	}
}
