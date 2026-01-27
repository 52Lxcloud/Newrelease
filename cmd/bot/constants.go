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

// Markdown 转义器
var markdownEscaper = strings.NewReplacer(
	"\\", "\\\\",
	"_", "\\_",
	"*", "\\*",
	"`", "\\`",
	"[", "\\[",
	"]", "\\]",
)

// 消息模板
const (
	startMessage               = "你好，管理员！请选择一个操作："
	repoPromptMessage          = "请发送你要监控的仓库。\n格式为 `owner/repository`（例如：`aiogram/aiogram`）。"
	cancelMessage              = "已取消设置。你可以发送 /start 重新开始。"
	listEmptyMessage           = "当前没有已添加的仓库。"
	listHeaderMessage          = "已添加的仓库："
	invalidRepoMessage         = "格式不正确。\n请使用 `owner/repository` 的格式后重试。"
	monitorTypePromptMessage   = "请选择要监控的类型："
	branchPromptMessage        = "请选择要监控的分支："
	branchCustomPromptMessage  = "请输入要监控的分支名称："
	channelPromptMessage       = "请选择通知方式："
	channelCustomPromptMessage = "请发送频道的用户名（例如：`@yourchannel`）。"
	channelAcceptedMessage     = "好的！现在，请把本机器人添加为你的 Telegram 频道*管理员*并授予「发布消息」权限。\n\n完成后，请发送频道的用户名（例如：`@yourchannel`）。"
	channelNotFoundMessage     = "找不到该频道。请检查用户名，并确保已添加机器人。"
	botNotAdminMessage         = "我还不是该频道的管理员。请确保机器人拥有"发布消息"权限后再试。"
	unexpectedErrorMessage     = "发生了未知错误，请稍后再试。"
	setupSuccessMessageTmpl    = "✅ 设置成功！\n\n*仓库*: `%s`\n*通知方式*: %s\n*监控类型*: %s\n%s"
	repoAcceptedMessageTmpl    = "好的！我将监控 `%s`。\n\n现在，请把本机器人添加为你的 Telegram 频道*管理员*并授予"发布消息"权限。\n\n完成后，请发送频道的用户名（例如：`@yourchannel`）。"
	releaseMessageTmpl         = "*新版本发布：%s*\n\n*仓库*: `%s`\n*标签*: `%s`\n\n[在 GitHub 查看 Release](%s)"
	commitMessageTmpl          = "*新提交*\n\n*仓库*: `%s`\n*分支*: `%s`\n*作者*: %s\n*信息*: %s\n*提交*: `%s`\n\n[查看提交](%s)"
	telegramParseModeMarkdown  = "Markdown"
)

// Callback 数据标识
const (
	callbackAddRepo        = "action:add_repo"
	callbackListRepos      = "action:list_repos"
	callbackCancel         = "action:cancel"
	callbackMonitorRelease = "monitor:release"
	callbackMonitorCommit  = "monitor:commit"
	callbackMonitorBoth    = "monitor:both"
	callbackBranchMain     = "branch:main"
	callbackBranchMaster   = "branch:master"
	callbackBranchCustom   = "branch:custom"
	callbackChannelPrivate = "channel:private"
	callbackChannelCustom  = "channel:custom"
)
