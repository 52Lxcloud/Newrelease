package main

import (
	"regexp"
	"strings"
	"time"
)

// 配置常量
const (
	configFile     = "/data/configs.json"
	checkInterval  = 60 * time.Second
	initialDelay   = 15 * time.Second
	repoCheckDelay = 2 * time.Second
	defaultBranch  = "main"
)

// 正则表达式
var repoRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+$`)

// Markdown 转义器 - Telegram Markdown 模式只需转义这几个字符
var markdownEscaper = strings.NewReplacer(
	"_", "\\_",  // 斜体标记
	"*", "\\*",  // 粗体标记
	"`", "\\`",  // 代码标记
	"[", "\\[",  // 链接标记
)

// 消息模板
const (
	listEmptyMessage           = "📭 当前没有已添加的仓库。\n\n使用 `/add owner/repo` 添加监控"
	listHeaderMessage          = "📚 *已监控的仓库*"
	invalidRepoMessage         = "❌ *格式错误*\n\n请使用 `owner/repository` 格式\n例如：`aiogram/aiogram`"
	repoExistsMessage          = "⚠️ *该仓库已存在相同配置*\n无需重复添加"
	deleteSuccessMessageTmpl   = "🗑️ *删除成功*\n\n已停止监控 `%s`"
	channelNotFoundMessage     = "❌ *频道不存在*\n\n请检查用户名，并确保已添加机器人为管理员"
	botNotAdminMessage         = "⚠️ *权限不足*\n\n请将机器人添加为频道管理员\n并授予 \"发布消息\" 权限"
	unexpectedErrorMessage     = "❌ *操作失败*\n\n发生未知错误，请稍后重试"
	setupSuccessMessageTmpl    = "✅ *添加成功*\n\n📦 *仓库*: `%s`\n📢 *通知*: %s\n🔍 *监控*: %s%s\n\n监控已启动，将在发现更新时通知你"
	
	// Release 通知模板 - 更醒目的样式
	releaseMessageTmpl = "*新版本*\n\n" +
		"📦 *%s*\n" +
		"└─ `%s`\n\n" +
		"🏷️ *版本*: `%s`\n\n" +
		"[查看详情](%s)"
	
	// Commit 通知模板 - 极简风格
	commitMessageTmpl = "*新提交*\n\n" +
		"📦 *%s*\n" +
		"└─ `%s`\n" +
		"└─ %s\n\n" +
		"```\n%s\n```\n\n" +
		"[查看详情](%s)"
	
	telegramParseModeMarkdown = "Markdown"
)
