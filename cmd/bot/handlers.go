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
	if cmd != "" {
		Logger.Debug("🔧 Command: %s", cmd)
	}
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
		if cmd != "" {
			Logger.Debug("⚠️ Unknown command: %s", cmd)
		}
	}
}

// handleStart 处理 /start 命令
func handleStart(tg *telegramClient, chatID int64) {
	tg.sendMessage(chatID, Messages.Help(), telegramParseModeMarkdown, false, "", 0)
}

// handleList 处理 /list 命令
func handleList(tg *telegramClient, chatID int64) {
	msg, err := buildRepoListMessage()
	if err != nil {
		log.Printf("Failed to build repo list: %v", err)
		tg.sendMessage(chatID, Messages.ErrorUnexpected(), telegramParseModeMarkdown, false, "", 0)
		return
	}
	tg.sendMessage(chatID, msg, telegramParseModeMarkdown, false, "", 0)
}

// handleAdd 处理 /add 命令
func handleAdd(tg *telegramClient, chatID int64, text string) {
	// 解析命令参数
	args := strings.Fields(text)
	if len(args) < 2 {
		tg.sendMessage(chatID, Messages.ErrorFormat(), telegramParseModeMarkdown, false, "", 0)
		return
	}

	repo := args[1]
	
	// 支持 owner/repo:branch 格式
	branch := ""
	if strings.Contains(repo, ":") {
		parts := strings.SplitN(repo, ":", 2)
		repo = parts[0]
		branch = parts[1]
	}

	if !repoRegexp.MatchString(repo) {
		tg.sendMessage(chatID, Messages.ErrorInvalidRepo(), telegramParseModeMarkdown, false, "", 0)
		return
	}

	monitorRelease := false
	monitorCommit := false
	chatTarget := "" // 可以是 @username 或群组 ID

	// 解析参数
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "-r":
			monitorRelease = true
		case "-c":
			monitorCommit = true
		default:
			// 支持 @username 格式
			if strings.HasPrefix(args[i], "@") {
				chatTarget = args[i]
			} else if strings.HasPrefix(args[i], "-") && len(args[i]) > 1 {
				// 支持群组 ID 格式（负数，如 -1003786162788）
				if _, err := strconv.ParseInt(args[i], 10, 64); err == nil {
					chatTarget = args[i]
				}
			}
		}
	}
	// 获取仓库信息（验证仓库存在并获取名称/默认分支）
	repoInfo, err := getRepoInfo(httpClient, repo)
	if err != nil {
		log.Printf("Failed to get repo info for %s: %v", repo, err)
		tg.sendMessage(chatID, Messages.ErrorInvalidRepo(), telegramParseModeMarkdown, false, "", 0)
		return
	}
	// 如果没有指定监控类型，默认两者都监控
	if !monitorRelease && !monitorCommit {
		monitorRelease = true
		monitorCommit = true
	}

	// 如果 branch 仍然为空，使用 GitHub 返回的默认分支
	if branch == "" {
		branch = repoInfo.DefaultBranch
	}

	// 处理频道/群组
	var channelID int64
	var channelTitle string
	var threadID int64 = 0
	var tgChat *chat
	
	if chatTarget != "" {
		c, err := tg.getChat(chatTarget)
		if err != nil {
			log.Printf("Failed to get chat %s: %v", chatTarget, err)
			tg.sendMessage(chatID, Messages.ErrorChannelNotFound(), telegramParseModeMarkdown, false, "", 0)
			return
		}
		
		// 检查机器人是否为管理员
		admins, err := tg.getChatAdministrators(c.ID)
		if err != nil {
			tg.sendMessage(chatID, Messages.ErrorBotNotAdmin(), telegramParseModeMarkdown, false, "", 0)
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
			tg.sendMessage(chatID, Messages.ErrorBotNotAdmin(), telegramParseModeMarkdown, false, "", 0)
			return
		}
		
		channelID = c.ID
		channelTitle = c.Title
		tgChat = c
	} else {
		channelTitle = "私聊"
	}

	// 加载现有配置
	configs, err := loadConfigs()
	if err != nil {
		log.Printf("Failed to load configs: %v", err)
		tg.sendMessage(chatID, Messages.ErrorUnexpected(), telegramParseModeMarkdown, false, "", 0)
		return
	}

	// 检查重复（在创建话题之前检查）
	// 如果由于没有 ThreadID 无法完全匹配，我们也应该检查该仓库是否已经在这个频道以相同的配置存在
	for _, cfg := range configs {
		if cfg.Repo == repo &&
			cfg.ChannelID == channelID &&
			cfg.MonitorRelease == monitorRelease &&
			cfg.MonitorCommit == monitorCommit &&
			cfg.Branch == branch {
			tg.sendMessage(chatID, Messages.ErrorRepoExists(), telegramParseModeMarkdown, false, "", 0)
			return
		}
	}

	// 如果是开启话题功能的群组，自动创建话题
	if tgChat != nil && tgChat.IsForum {
		topicName := repoInfo.Name
		topic, err := tg.createForumTopic(tgChat.ID, topicName)
		if err != nil {
			log.Printf("Failed to create forum topic for %s: %v", repo, err)
			tg.sendMessage(chatID, Messages.ErrorCreateTopic(), telegramParseModeMarkdown, false, "", 0)
			return
		}
		threadID = topic.MessageThreadID
		log.Printf("📝 Created topic '%s' (thread_id: %d) in %s", topicName, threadID, channelTitle)
	}

	// 创建新配置
	newConfig := repoConfig{
		Repo:           repo,
		RepoName:       repoInfo.Name,
		ChannelID:      channelID,
		ChannelTitle:   channelTitle,
		ThreadID:       threadID,
		MonitorRelease: monitorRelease,
		MonitorCommit:  monitorCommit,
		Branch:         branch,
	}

	// 添加并保存
	configs = append(configs, newConfig)
	if err := saveConfigs(configs); err != nil {
		log.Printf("Failed to save configs: %v", err)
		tg.sendMessage(chatID, Messages.ErrorUnexpected(), telegramParseModeMarkdown, false, "", 0)
		return
	}

	// 构建成功消息
	var notifyWay string
	if threadID > 0 {
		// 群组 + 话题
		notifyWay = fmt.Sprintf("%s \\> %s", MDV2.Escape(channelTitle), MDV2.Escape(repoInfo.Name))
	} else if channelID != 0 {
		// 频道/群组
		notifyWay = MDV2.Escape(channelTitle)
	} else {
		// 私聊
		notifyWay = "私聊"
	}

	monitorTypeStr := ""
	if monitorRelease && monitorCommit {
		monitorTypeStr = "Release \\+ Commit"
	} else if monitorRelease {
		monitorTypeStr = "Release"
	} else if monitorCommit {
		monitorTypeStr = "Commit"
	}

	branchInfo := ""
	if monitorCommit {
		branchInfo = branch
	}

	successMsg := Messages.SuccessAdded(
		MDV2.Escape(repo),
		notifyWay,
		monitorTypeStr,
		branchInfo,
	)

	tg.sendMessage(chatID, successMsg, telegramParseModeMarkdown, false, "", 0)
	if threadID > 0 {
		log.Printf("➕ Added: %s -> %s (topic: %d)", repo, channelTitle, threadID)
	} else {
		log.Printf("➕ Added: %s", repo)
	}
}

// handleDelete 处理 /delete 命令
func handleDelete(tg *telegramClient, chatID int64, text string) {
	args := strings.Fields(text)
	if len(args) < 2 {
		tg.sendMessage(chatID, Messages.ErrorDeleteFormat(), telegramParseModeMarkdown, false, "", 0)
		return
	}

	index, err := strconv.Atoi(args[1])
	if err != nil || index < 1 {
		tg.sendMessage(chatID, "❌ 序号必须是大于 0 的数字！", "", false, "", 0)
		return
	}

	configs, err := loadConfigs()
	if err != nil {
		log.Printf("Failed to load configs: %v", err)
		tg.sendMessage(chatID, Messages.ErrorUnexpected(), telegramParseModeMarkdown, false, "", 0)
		return
	}

	if index > len(configs) {
		tg.sendMessage(chatID, fmt.Sprintf("❌ 序号超出范围！当前只有 %d 个仓库。", len(configs)), "", false, "", 0)
		return
	}

	// 删除配置
	deletedRepo := configs[index-1].Repo
	configs = append(configs[:index-1], configs[index:]...)

	if err := saveConfigs(configs); err != nil {
		log.Printf("Failed to save configs: %v", err)
		tg.sendMessage(chatID, Messages.ErrorUnexpected(), telegramParseModeMarkdown, false, "", 0)
		return
	}

	successMsg := Messages.SuccessDeleted(MDV2.Escape(deletedRepo))
	tg.sendMessage(chatID, successMsg, telegramParseModeMarkdown, false, "", 0)
	log.Printf("🗑️ Deleted: %s", deletedRepo)
}
