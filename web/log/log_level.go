package log

import (
	"fmt"
	"io"
	"os"
	"time"
)

type LoggerLevel int //级别

const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

// Logger 日志结构
type Logger struct {
	Formatter LoggerFormatterSplit
	Outs      []io.Writer
	Level     LoggerLevel
}

type LoggerFormatterSplit struct {
	Color bool
	Level LoggerLevel
}

func New() *Logger {
	return &Logger{}
}

func (l *Logger) Info(msg any) {
	l.Print(LevelInfo, msg)
}

func (l *Logger) Debug(msg any) {
	l.Print(LevelDebug, msg)
}

func (l *Logger) Error(msg any) {
	l.Print(LevelError, msg)
}

func (l *Logger) Print(level LoggerLevel, msg any) {
	if l.Level > level {
		//级别不满足 不打印日志
		return
	}
	l.Formatter.Level = level
	formatter := l.Formatter.formatter(msg)
	for _, out := range l.Outs {
		_, err := fmt.Fprint(out, formatter)
		if err != nil {
			return
		}
	}
}

func (f *LoggerFormatterSplit) formatter(msg any) string {
	now := time.Now()
	return fmt.Sprintf("[msgo] %v | level=%s | msg=%#v \n",
		now.Format("2006/01/02 - 15:04:05"),
		f.Level.Level(), msg,
	)
}

func (level LoggerLevel) Level() string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

func Default() *Logger {
	logger := New()
	out := os.Stdout
	logger.Outs = append(logger.Outs, out)
	logger.Level = LevelDebug
	logger.Formatter = LoggerFormatterSplit{}
	return logger
}
