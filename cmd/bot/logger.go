package main

import (
	"log"
	"os"
	"strings"
)

// Logger 日志工具
var Logger = &logger{
	debug: strings.ToLower(os.Getenv("DEBUG")) == "true" || os.Getenv("DEBUG") == "1",
}

type logger struct {
	debug bool
}

// Debug 调试日志（仅当 DEBUG=true 时输出）
func (l *logger) Debug(format string, v ...interface{}) {
	if l.debug {
		log.Printf(format, v...)
	}
}

// Info 普通信息日志（始终输出）
func (l *logger) Info(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Error 错误日志（始终输出）
func (l *logger) Error(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// IsDebug 检查是否开启 debug 模式
func (l *logger) IsDebug() bool {
	return l.debug
}
