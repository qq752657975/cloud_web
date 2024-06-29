package tracer

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"io"
	"net/http"
)

// CreateTracer 创建一个新的 Jaeger Tracer 单机使用
// serviceName: 服务名称
// samplerConfig: 采样配置
// reporter: 报告配置
// options: 其他可选配置
func CreateTracer(serviceName string, samplerConfig *config.SamplerConfig, reporter *config.ReporterConfig, options ...config.Option) (opentracing.Tracer, io.Closer, error) {
	var cfg = config.Configuration{
		ServiceName: serviceName,   // 服务名称
		Sampler:     samplerConfig, // 采样配置
		Reporter:    reporter,      // 报告配置
	}
	tracer, closer, err := cfg.NewTracer(options...) // 创建 Tracer
	return tracer, closer, err
}

// CreateTracerHeader 创建一个新的 Jaeger Tracer 并从 HTTP Header 中提取 SpanContext 分布式使用
// serviceName: 服务名称
// header: HTTP 请求头
// samplerConfig: 采样配置
// reporter: 报告配置
// options: 其他可选配置
func CreateTracerHeader(serviceName string, header http.Header, samplerConfig *config.SamplerConfig, reporter *config.ReporterConfig, options ...config.Option) (opentracing.Tracer, io.Closer, opentracing.SpanContext, error) {
	var cfg = config.Configuration{
		ServiceName: serviceName,   // 服务名称
		Sampler:     samplerConfig, // 采样配置
		Reporter:    reporter,      // 报告配置
	}
	tracer, closer, err := cfg.NewTracer(options...) // 创建 Tracer
	// 继承别的进程传递过来的上下文
	spanContext, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(header)) // 从 HTTP Header 中提取 SpanContext

	return tracer, closer, spanContext, err
}
