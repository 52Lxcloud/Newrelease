package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// handleMessage 处理文本消息
func handleMessage(tg *telegramClient, msg *message, adminID int64) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	cmd := parseCommand(text)
	switch cmd {
	case "/start", "/help":
		handleStart(tg, msg.Chat.ID)
	case "/list":
		handleList(tg, msg.Chat.ID)
	case "/add":
		handleAdd(tg, msg.Chat.ID, text)
	case "/delete", "/del", "/remove":
		handleDelete(tg, msg.Chat.ID, text)
	default:
		// 未知命令，不回复
	}
}

// handleStart 处理 /start 命令
func handleStart(tg *telegramClient, chatID int64) {
	helpText := "*GitHub Release & Commit 监控机器人*\n\n" +
		"*可用命令：*\n\n" +
		"• `/list` - 查看所有监控的仓库\n\n" +
		"• `/add` - 添加仓库监控\n" +
		"  格式：`/add owner/repo[:branch] [选项]`\n  \n" +
		"  选项：\n" +
		"  `-r` : 监控 Release\n" +
		"  `-c` : 监控 Commit\n" +
		"  `@channel` : 发送到指定频道（默认私聊）\n  \n" +
		"  示例：\n" +
		"  `/add nginx/nginx:master -r`\n" +
		"  `/add golang/go:dev -c`\n" +
		"  `/add facebook/react`\n\n" +
		"• `/delete <序号>` - 删除监控\n" +
		"  示例：`/delete 1`\n\n" +
		"*提示：*\n" +
		"• 默认监控 Release 和 Commit\n" +
		"• 用 `:branch` 快速指定其他分支\n" +
		"• 频道需先添加机器人为管理员"

	tg.sendMessage(chatID, helpText, telegramParseModeMarkdown, false, "")
}

// handleList 处理 /list 命令
func handleList(tg *telegramClient, chatID int64) {
	msg, err := buildRepoListMessage()
	if err != nil {
		log.Printf("Failed to build repo list: %v", err)
		tg.sendMessage(chatID, unexpectedErrorMessage, "", false, "")
		return
	}
	tg.sendMessage(chatID, msg, telegramParseModeMarkdown, false, "")
}

// handleAdd 处理 /add 命令
func handleAdd(tg *telegramClient, chatID int64, text string) {
	// 解析命令参数
	args := strings.Fields(text)
	if len(args) < 2 {
		tg.sendMessage(chatID, "❌ 格式错误！\n\n使用方法：`/add owner/repo [选项]`\n\n发送 `/start` 查看详细帮助。", telegramParseModeMarkdown, false, "")
		return
	}

	repo := args[1]
	
	// 支持 owner/repo:branch 格式
	branch := "" // Initialize branch here
	if strings.Contains(repo, ":") {
		parts := strings.SplitN(repo, ":", 2)
		repo = parts[0]
		branch = parts[1]
	}

	if !repoRegexp.MatchString(repo) {
		tg.sendMessage(chatID, invalidRepoMessage, telegramParseModeMarkdown, false, "")
		return
	}

	monitorRelease := false
	monitorCommit := false
	channelUsername := ""

	// 解析参数
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "-r":
			monitorRelease = true
		case "-c":
			monitorCommit = true
		default:
			if strings.HasPrefix(args[i], "@") {
				channelUsername = args[i]
			}
		}
	}
	// 如果没有指定监控类型，默认两者都监控
	if !monitorRelease && !monitorCommit {
		monitorRelease = true
		monitorCommit = true
	}

	// 如果 branch 仍然为空，则从 GitHub 获取默认分支
	if branch == "" && monitorCommit {
		defaultBr, err := getRepoDefaultBranch(httpClient, repo)
		if err != nil {
			log.Printf("Failed to get default branch for %s: %v, using 'main'", repo, err)
			branch = "main"
		} else {
			branch = defaultBr
		}
	}

	// 处理频道
	var channelID int64
	var channelTitle string
	if channelUsername != "" {
		chat, err := tg.getChat(channelUsername)
		if err != nil {
			log.Printf("Failed to get chat %s: %v", channelUsername, err)
			tg.sendMessage(chatID, channelNotFoundMessage, telegramParseModeMarkdown, false, "")
			return
		}
		
		// 检查机器人是否为管理员
		admins, err := tg.getChatAdministrators(chat.ID)
		if err != nil {
			tg.sendMessage(chatID, botNotAdminMessage, telegramParseModeMarkdown, false, "")
			return
		}
		
		isAdmin := false
		for _, admin := range admins {
			if admin.User.ID == tg.botID {
				isAdmin = true
				break
			}
		}
		
		if !isAdmin {
			tg.sendMessage(chatID, botNotAdminMessage, telegramParseModeMarkdown, false, "")
			return
		}
		
		channelID = chat.ID
		channelTitle = chat.Title
	} else {
		channelTitle = "私聊"
	}

	// 加载现有配置
	configs, err := loadConfigs()
	if err != nil {
		log.Printf("Failed to load configs: %v", err)
		tg.sendMessage(chatID, unexpectedErrorMessage, "", false, "")
		return
	}

	// 创建新配置
	newConfig := repoConfig{
		Repo:           repo,
		ChannelID:      channelID,
		ChannelTitle:   channelTitle,
		MonitorRelease: monitorRelease,
		MonitorCommit:  monitorCommit,
		Branch:         branch,
	}

	// 检查重复
	if isDuplicateConfig(configs, newConfig) {
		tg.sendMessage(chatID, repoExistsMessage, telegramParseModeMarkdown, false, "")
		return
	}

	// 添加并保存
	configs = append(configs, newConfig)
	if err := saveConfigs(configs); err != nil {
		log.Printf("Failed to save configs: %v", err)
		tg.sendMessage(chatID, unexpectedErrorMessage, "", false, "")
		return
	}

	// 构建成功消息
	notifyWay := channelTitle
	if channelTitle == "" {
		notifyWay = "私聊"
	}

	monitorTypeStr := ""
	if monitorRelease && monitorCommit {
		monitorTypeStr = "Release + Commit"
	} else if monitorRelease {
		monitorTypeStr = "Release"
	} else if monitorCommit {
		monitorTypeStr = "Commit"
	}

	branchInfo := ""
	if monitorCommit {
		branchInfo = fmt.Sprintf("\n*分支*: `%s`", branch)
	}

	successMsg := fmt.Sprintf(setupSuccessMessageTmpl,
		escapeMarkdown(repo),
		escapeMarkdown(notifyWay),
		monitorTypeStr,
		branchInfo,
	)

	tg.sendMessage(chatID, successMsg, telegramParseModeMarkdown, false, "")
	log.Printf("Added repo: %s (Release: %v, Commit: %v, Branch: %s, Channel: %s)",
		repo, monitorRelease, monitorCommit, branch, notifyWay)
}

// handleDelete 处理 /delete 命令
func handleDelete(tg *telegramClient, chatID int64, text string) {
	args := strings.Fields(text)
	if len(args) < 2 {
		tg.sendMessage(chatID, "❌ 格式错误！\n\n使用方法：`/delete <序号>`\n\n先用 `/list` 查看序号。", telegramParseModeMarkdown, false, "")
		return
	}

	index, err := strconv.Atoi(args[1])
	if err != nil || index < 1 {
		tg.sendMessage(chatID, "❌ 序号必须是大于 0 的数字！", "", false, "")
		return
	}

	configs, err := loadConfigs()
	if err != nil {
		log.Printf("Failed to load configs: %v", err)
		tg.sendMessage(chatID, unexpectedErrorMessage, "", false, "")
		return
	}

	if index > len(configs) {
		tg.sendMessage(chatID, fmt.Sprintf("❌ 序号超出范围！当前只有 %d 个仓库。", len(configs)), "", false, "")
		return
	}

	// 删除配置
	deletedRepo := configs[index-1].Repo
	configs = append(configs[:index-1], configs[index:]...)

	if err := saveConfigs(configs); err != nil {
		log.Printf("Failed to save configs: %v", err)
		tg.sendMessage(chatID, unexpectedErrorMessage, "", false, "")
		return
	}

	successMsg := fmt.Sprintf(deleteSuccessMessageTmpl, escapeMarkdown(deletedRepo))
	tg.sendMessage(chatID, successMsg, telegramParseModeMarkdown, false, "")
	log.Printf("Deleted repo: %s", deletedRepo)
}

// isDuplicateConfig 检查是否存在重复配置
func isDuplicateConfig(configs []repoConfig, newConfig repoConfig) bool {
	for _, cfg := range configs {
		if cfg.Repo == newConfig.Repo &&
			cfg.ChannelID == newConfig.ChannelID &&
			cfg.MonitorRelease == newConfig.MonitorRelease &&
			cfg.MonitorCommit == newConfig.MonitorCommit &&
			cfg.Branch == newConfig.Branch {
			return true
		}
	}
	return false
}
