package log

import (
	"fmt"
	"github.com/ygb616/web"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

var IsDisplayColor = true
var DefaultWriter io.Writer = os.Stdout

type LogFormatterParams struct {
	Request        *http.Request
	TimeStamp      time.Time
	StatusCode     int
	Latency        time.Duration
	ClientIP       net.IP
	Method         string
	Path           string
	IsDisplayColor bool
}

type LoggerFormatter func(params LogFormatterParams) string

var defaultLogFormatter = func(params LogFormatterParams) string {
	statusCodeColor := params.StatusCodeColor()
	resetColor := params.ResetColor()
	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}
	if params.IsDisplayColor {
		return fmt.Sprintf("%s [web] %s |%s %v %s| %s %3d %s |%s %13v %s| %15s  |%s %-7s %s  %s %#v %s \n",
			yellow, resetColor, blue, params.TimeStamp.Format("2006/01/02 - 15:04:05"), resetColor,
			statusCodeColor, params.StatusCode, resetColor,
			red, params.Latency, resetColor,
			params.ClientIP,
			magenta, params.Method, resetColor,
			cyan, params.Path, resetColor,
		)
	}
	return fmt.Sprintf("[msgo] %v | %3d | %13v | %15s |%-7s %#v \n",
		params.TimeStamp.Format("2006/01/02 - 15:04:05"),
		params.StatusCode,
		params.Latency, params.ClientIP, params.Method, params.Path,
	)
}

type LoggingConfig struct {
	Formatter LoggerFormatter
	out       io.Writer
}

func LoggingWithConfig(conf LoggingConfig, next web.HandlerFunc) web.HandlerFunc {
	_ = fmt.Sprintf("%#v", red)
	formatter := conf.Formatter
	if formatter == nil {
		formatter = defaultLogFormatter
	}
	out := conf.out
	if out == nil {
		out = DefaultWriter
	}
	return func(ctx *web.Context) {
		param := LogFormatterParams{
			Request:        ctx.R,
			IsDisplayColor: IsDisplayColor,
		}
		// Start timer
		start := time.Now()
		path := ctx.R.URL.Path
		raw := ctx.R.URL.RawQuery
		//执行业务
		next(ctx)
		// stop timer
		stop := time.Now()
		latency := stop.Sub(start)
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(ctx.R.RemoteAddr))
		clientIP := net.ParseIP(ip)
		method := ctx.R.Method
		statusCode := ctx.StatusCode

		if raw != "" {
			path = path + "?" + raw
		}

		param.ClientIP = clientIP
		param.TimeStamp = stop
		param.Latency = latency
		param.StatusCode = statusCode
		param.Method = method
		param.Path = path
		_, err := fmt.Fprint(out, formatter(param))
		if err != nil {
			fmt.Printf("Logging error: %v\n", err)
		}
	}
}

func (p *LogFormatterParams) ResetColor() string {
	return reset
}

func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch code {
	case http.StatusOK:
		return green
	default:
		return red
	}
}

func Logging(next web.HandlerFunc) web.HandlerFunc {
	return LoggingWithConfig(LoggingConfig{}, next)
}
