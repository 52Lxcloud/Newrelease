package main

import (
	"regexp"
	"time"
)

// 配置常量
const (
	configFile     = "/data/configs.json"
	checkInterval  = 60 * time.Second
	initialDelay   = 15 * time.Second
	repoCheckDelay = 2 * time.Second
)

// 正则表达式
var repoRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+$`)

// Telegram 解析模式
const (
	telegramParseModeMarkdown = "MarkdownV2"
)
