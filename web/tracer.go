package web

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go/config"
	tracer2 "github.com/ygb616/web/tracer"
)

// Tracer 创建一个 Jaeger Tracer 的中间件函数
// serviceName: 服务名称
// samplerConfig: 采样配置
// reporter: 报告配置
// options: 其他可选配置
func Tracer(serviceName string, samplerConfig *config.SamplerConfig, reporter *config.ReporterConfig, options ...config.Option) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			// 接收 Jaeger 的信息，解析上下文
			// 使用 opentracing.GlobalTracer() 获取全局 Tracer
			tracer, closer, spanContext, _ := tracer2.CreateTracerHeader(serviceName, ctx.R.Header, samplerConfig, reporter, options...)
			defer closer.Close() // 确保在函数结束时关闭 Tracer

			// 生成依赖关系，并新建一个 span
			// 生成了 References []SpanReference 依赖关系
			startSpan := tracer.StartSpan(ctx.R.URL.Path, ext.RPCServerOption(spanContext))
			defer startSpan.Finish() // 确保在函数结束时结束 span

			// 记录 tag
			// 记录请求 URL
			ext.HTTPUrl.Set(startSpan, ctx.R.URL.Path)
			// 记录 HTTP 方法
			ext.HTTPMethod.Set(startSpan, ctx.R.Method)
			// 记录组件名称
			ext.Component.Set(startSpan, "Msgo-Http")

			// 在 header 中加上当前进程的上下文信息
			ctx.R = ctx.R.WithContext(opentracing.ContextWithSpan(ctx.R.Context(), startSpan))

			// 调用下一个处理函数
			next(ctx)

			// 继续设置 tag
			ext.HTTPStatusCode.Set(startSpan, uint16(ctx.StatusCode))
		}
	}
}
