package main

import (
	"fmt"
	"log"
	"strings"
)

// handleCallbackQuery 处理回调查询
func handleCallbackQuery(tg *telegramClient, cb *callbackQuery, adminID int64) {
	chatID := int64(0)
	messageID := 0
	if cb.Message != nil && cb.Message.Chat != nil {
		chatID = cb.Message.Chat.ID
		messageID = cb.Message.MessageID
	}

	switch cb.Data {
	case callbackAddRepo:
		handleAddRepo(tg, cb, chatID, messageID)
	case callbackMonitorRelease:
		handleMonitorRelease(tg, cb, chatID, messageID)
	case callbackMonitorCommit:
		handleMonitorCommit(tg, cb, chatID, messageID)
	case callbackMonitorBoth:
		handleMonitorBoth(tg, cb, chatID, messageID)
	case callbackBranchMain:
		handleBranchMain(tg, cb, chatID, messageID)
	case callbackBranchMaster:
		handleBranchMaster(tg, cb, chatID, messageID)
	case callbackBranchCustom:
		handleBranchCustom(tg, cb, chatID, messageID)
	case callbackChannelPrivate:
		handleChannelPrivate(tg, cb, chatID, messageID)
	case callbackChannelCustom:
		handleChannelCustom(tg, cb, chatID, messageID)
	case callbackListRepos:
		handleListRepos(tg, cb, chatID, messageID)
	case callbackCancel:
		handleCancel(tg, cb, chatID, messageID)
	default:
		_ = tg.answerCallbackQuery(cb.ID, "", false)
	}
}

func handleAddRepo(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	setSession(setupSession{state: stateWaitingRepo, lastBotMsgID: messageID, chatID: chatID})
	if chatID != 0 && messageID != 0 {
		if err := tg.editMessageText(chatID, messageID, repoPromptMessage, telegramParseModeMarkdown, false, cancelKeyboard()); err != nil {
			log.Printf("Failed to edit repo prompt message: %v", err)
		}
	}
	_ = tg.answerCallbackQuery(cb.ID, "", false)
}

func handleMonitorRelease(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	sess := getSession()
	if sess.state == stateWaitingMonitorType {
		sess.state = stateWaitingChannelType
		sess.monitorRelease = true
		sess.monitorCommit = false
		sess.lastBotMsgID = messageID
		sess.chatID = chatID
		setSession(sess)
		if chatID != 0 && messageID != 0 {
			if err := tg.editMessageText(chatID, messageID, channelPromptMessage, telegramParseModeMarkdown, false, channelKeyboard()); err != nil {
				log.Printf("Failed to edit channel prompt message: %v", err)
			}
		}
		_ = tg.answerCallbackQuery(cb.ID, "", false)
	}
}

func handleMonitorCommit(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	sess := getSession()
	if sess.state == stateWaitingMonitorType {
		sess.state = stateWaitingBranch
		sess.monitorRelease = false
		sess.monitorCommit = true
		sess.lastBotMsgID = messageID
		sess.chatID = chatID
		setSession(sess)
		if chatID != 0 && messageID != 0 {
			if err := tg.editMessageText(chatID, messageID, branchPromptMessage, telegramParseModeMarkdown, false, branchKeyboard()); err != nil {
				log.Printf("Failed to edit branch prompt message: %v", err)
			}
		}
		_ = tg.answerCallbackQuery(cb.ID, "", false)
	}
}

func handleMonitorBoth(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	sess := getSession()
	if sess.state == stateWaitingMonitorType {
		sess.state = stateWaitingBranch
		sess.monitorRelease = true
		sess.monitorCommit = true
		sess.lastBotMsgID = messageID
		sess.chatID = chatID
		setSession(sess)
		if chatID != 0 && messageID != 0 {
			if err := tg.editMessageText(chatID, messageID, branchPromptMessage, telegramParseModeMarkdown, false, branchKeyboard()); err != nil {
				log.Printf("Failed to edit branch prompt message: %v", err)
			}
		}
		_ = tg.answerCallbackQuery(cb.ID, "", false)
	}
}

func handleBranchMain(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	handleBranchSelection(tg, cb, chatID, messageID, "main")
}

func handleBranchMaster(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	handleBranchSelection(tg, cb, chatID, messageID, "master")
}

func handleBranchSelection(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int, branch string) {
	sess := getSession()
	if sess.state == stateWaitingBranch {
		sess.state = stateWaitingChannelType
		sess.branch = branch
		sess.lastBotMsgID = messageID
		sess.chatID = chatID
		setSession(sess)
		if chatID != 0 && messageID != 0 {
			if err := tg.editMessageText(chatID, messageID, channelPromptMessage, telegramParseModeMarkdown, false, channelKeyboard()); err != nil {
				log.Printf("Failed to edit channel prompt message: %v", err)
			}
		}
		_ = tg.answerCallbackQuery(cb.ID, "", false)
	}
}

func handleBranchCustom(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	sess := getSession()
	if sess.state == stateWaitingBranch {
		sess.state = stateWaitingBranchCustom
		sess.lastBotMsgID = messageID
		sess.chatID = chatID
		setSession(sess)
		if chatID != 0 && messageID != 0 {
			if err := tg.editMessageText(chatID, messageID, branchCustomPromptMessage, telegramParseModeMarkdown, false, cancelKeyboard()); err != nil {
				log.Printf("Failed to edit branch custom prompt message: %v", err)
			}
		}
		_ = tg.answerCallbackQuery(cb.ID, "", false)
	}
}

func handleChannelPrivate(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	sess := getSession()
	if sess.state == stateWaitingChannelType {
		configs, err := loadConfigs()
		if err != nil {
			log.Printf("Failed to load configs: %v", err)
			if chatID != 0 && messageID != 0 {
				if err := tg.editMessageText(chatID, messageID, unexpectedErrorMessage, "", false, startKeyboard()); err != nil {
					log.Printf("Failed to edit error message: %v", err)
				}
			}
			_ = tg.answerCallbackQuery(cb.ID, "", false)
			return
		}

		newConfig := repoConfig{
			Repo:           sess.repo,
			ChannelID:      0,
			ChannelTitle:   "私聊",
			MonitorRelease: sess.monitorRelease,
			MonitorCommit:  sess.monitorCommit,
			Branch:         sess.branch,
			LastReleaseID:  nil,
			LastCommitSHA:  nil,
		}
		configs = append(configs, newConfig)
		if err := saveConfigs(configs); err != nil {
			log.Printf("Failed to save configs: %v", err)
			if chatID != 0 && messageID != 0 {
				if err := tg.editMessageText(chatID, messageID, unexpectedErrorMessage, "", false, startKeyboard()); err != nil {
					log.Printf("Failed to edit error message: %v", err)
				}
			}
			_ = tg.answerCallbackQuery(cb.ID, "", false)
			return
		}

		successMessage := buildSuccessMessage(sess, "私聊")
		if chatID != 0 && messageID != 0 {
			if err := tg.editMessageText(chatID, messageID, successMessage, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
				log.Printf("Failed to edit success message: %v", err)
			}
		}
		setSession(setupSession{state: stateIdle})
		_ = tg.answerCallbackQuery(cb.ID, "", false)
	}
}

func handleChannelCustom(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	sess := getSession()
	if sess.state == stateWaitingChannelType {
		sess.state = stateWaitingChannel
		sess.lastBotMsgID = messageID
		sess.chatID = chatID
		setSession(sess)
		if chatID != 0 && messageID != 0 {
			if err := tg.editMessageText(chatID, messageID, channelCustomPromptMessage, telegramParseModeMarkdown, false, cancelKeyboard()); err != nil {
				log.Printf("Failed to edit channel custom prompt message: %v", err)
			}
		}
		_ = tg.answerCallbackQuery(cb.ID, "", false)
	}
}

func handleListRepos(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	if chatID != 0 && messageID != 0 {
		messageText, err := buildRepoListMessage()
		if err != nil {
			log.Printf("Failed to build repo list: %v", err)
			if err := tg.editMessageText(chatID, messageID, unexpectedErrorMessage, "", false, startKeyboard()); err != nil {
				log.Printf("Failed to edit repo list error message: %v", err)
			}
		} else {
			if err := tg.editMessageText(chatID, messageID, messageText, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
				log.Printf("Failed to edit repo list message: %v", err)
			}
		}
	}
	_ = tg.answerCallbackQuery(cb.ID, "", false)
}

func handleCancel(tg *telegramClient, cb *callbackQuery, chatID int64, messageID int) {
	setSession(setupSession{state: stateIdle})
	if chatID != 0 && messageID != 0 {
		if err := tg.editMessageText(chatID, messageID, cancelMessage, "", false, startKeyboard()); err != nil {
			log.Printf("Failed to edit cancel message: %v", err)
		}
	}
	_ = tg.answerCallbackQuery(cb.ID, "", false)
}

// handleMessage 处理文本消息
func handleMessage(tg *telegramClient, msg *message, adminID int64) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	switch parseCommand(text) {
	case "/start":
		setSession(setupSession{state: stateIdle})
		if _, err := tg.sendMessage(msg.Chat.ID, startMessage, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
			log.Printf("Failed to send start message: %v", err)
		}
		return
	case "/list":
		messageText, err := buildRepoListMessage()
		if err != nil {
			log.Printf("Failed to build repo list: %v", err)
			if _, err := tg.sendMessage(msg.Chat.ID, unexpectedErrorMessage, "", false, startKeyboard()); err != nil {
				log.Printf("Failed to send repo list error message: %v", err)
			}
		} else {
			if _, err := tg.sendMessage(msg.Chat.ID, messageText, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
				log.Printf("Failed to send repo list message: %v", err)
			}
		}
		return
	case "/cancel":
		setSession(setupSession{state: stateIdle})
		if _, err := tg.sendMessage(msg.Chat.ID, cancelMessage, "", false, startKeyboard()); err != nil {
			log.Printf("Failed to send cancel message: %v", err)
		}
		return
	}

	sess := getSession()
	switch sess.state {
	case stateWaitingRepo:
		handleWaitingRepo(tg, msg, sess, text)
	case stateWaitingBranchCustom:
		handleWaitingBranchCustom(tg, msg, sess, text)
	case stateWaitingChannel:
		handleWaitingChannel(tg, msg, sess, text)
	}
}

func handleWaitingRepo(tg *telegramClient, msg *message, sess setupSession, text string) {
	// 删除之前的提示消息
	if sess.lastBotMsgID != 0 && sess.chatID != 0 {
		_ = tg.deleteMessage(sess.chatID, sess.lastBotMsgID)
	}

	if !repoRegexp.MatchString(text) {
		_ = tg.deleteMessage(msg.Chat.ID, msg.MessageID)
		newMsg, err := tg.sendMessage(msg.Chat.ID, invalidRepoMessage, telegramParseModeMarkdown, false, cancelKeyboard())
		if err != nil {
			log.Printf("Failed to send invalid repo message: %v", err)
		} else {
			sess.lastBotMsgID = newMsg.MessageID
			sess.chatID = msg.Chat.ID
			setSession(sess)
		}
		return
	}

	sess.state = stateWaitingMonitorType
	sess.repo = text
	setSession(sess)
	if _, err := tg.sendMessage(msg.Chat.ID, monitorTypePromptMessage, telegramParseModeMarkdown, false, monitorTypeKeyboard()); err != nil {
		log.Printf("Failed to send monitor type prompt message: %v", err)
	}
}

func handleWaitingBranchCustom(tg *telegramClient, msg *message, sess setupSession, text string) {
	branch := strings.TrimSpace(text)
	if branch == "" {
		branch = defaultBranch
	}
	sess.state = stateWaitingChannelType
	sess.branch = branch
	setSession(sess)
	if _, err := tg.sendMessage(msg.Chat.ID, channelPromptMessage, telegramParseModeMarkdown, false, channelKeyboard()); err != nil {
		log.Printf("Failed to send channel prompt message: %v", err)
	}
}

func handleWaitingChannel(tg *telegramClient, msg *message, sess setupSession, text string) {
	// 删除之前的提示消息
	if sess.lastBotMsgID != 0 && sess.chatID != 0 {
		_ = tg.deleteMessage(sess.chatID, sess.lastBotMsgID)
	}

	channelName := text
	tgChat, err := tg.getChat(channelName)
	if err != nil {
		newMsg, err := tg.sendMessage(msg.Chat.ID, channelNotFoundMessage, "", false, cancelKeyboard())
		if err != nil {
			log.Printf("Failed to send channel not found message: %v", err)
		} else {
			sess.lastBotMsgID = newMsg.MessageID
			sess.chatID = msg.Chat.ID
			setSession(sess)
		}
		return
	}

	admins, err := tg.getChatAdministrators(tgChat.ID)
	if err != nil {
		log.Printf("Error validating channel %s: %v", channelName, err)
		newMsg, err := tg.sendMessage(msg.Chat.ID, unexpectedErrorMessage, "", false, cancelKeyboard())
		if err != nil {
			log.Printf("Failed to send unexpected error message: %v", err)
		} else {
			sess.lastBotMsgID = newMsg.MessageID
			sess.chatID = msg.Chat.ID
			setSession(sess)
		}
		return
	}

	isBotAdmin := false
	for _, admin := range admins {
		if admin.User.ID == tg.botID {
			isBotAdmin = true
			break
		}
	}
	if !isBotAdmin {
		newMsg, err := tg.sendMessage(msg.Chat.ID, botNotAdminMessage, "", false, cancelKeyboard())
		if err != nil {
			log.Printf("Failed to send bot not admin message: %v", err)
		} else {
			sess.lastBotMsgID = newMsg.MessageID
			sess.chatID = msg.Chat.ID
			setSession(sess)
		}
		return
	}

	configs, err := loadConfigs()
	if err != nil {
		log.Printf("Failed to load configs: %v", err)
		if _, err := tg.sendMessage(msg.Chat.ID, unexpectedErrorMessage, "", false, cancelKeyboard()); err != nil {
			log.Printf("Failed to send unexpected error message: %v", err)
		}
		return
	}

	newConfig := repoConfig{
		Repo:           sess.repo,
		ChannelID:      tgChat.ID,
		ChannelTitle:   tgChat.Title,
		MonitorRelease: sess.monitorRelease,
		MonitorCommit:  sess.monitorCommit,
		Branch:         sess.branch,
		LastReleaseID:  nil,
		LastCommitSHA:  nil,
	}
	configs = append(configs, newConfig)
	if err := saveConfigs(configs); err != nil {
		log.Printf("Failed to save configs: %v", err)
		if _, err := tg.sendMessage(msg.Chat.ID, unexpectedErrorMessage, "", false, cancelKeyboard()); err != nil {
			log.Printf("Failed to send unexpected error message: %v", err)
		}
		return
	}

	successMessage := buildSuccessMessage(sess, escapeMarkdown(tgChat.Title))
	if _, err := tg.sendMessage(msg.Chat.ID, successMessage, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
		log.Printf("Failed to send success message: %v", err)
	}
	setSession(setupSession{state: stateIdle})
}

// buildSuccessMessage 构建设置成功消息
func buildSuccessMessage(sess setupSession, channelTitle string) string {
	var monitorTypeDesc string
	if sess.monitorRelease && sess.monitorCommit {
		monitorTypeDesc = "Release + Commit"
	} else if sess.monitorRelease {
		monitorTypeDesc = "Release"
	} else {
		monitorTypeDesc = "Commit"
	}

	branchInfo := ""
	if sess.monitorCommit {
		branch := sess.branch
		if branch == "" {
			branch = defaultBranch
		}
		branchInfo = fmt.Sprintf("*分支*: `%s`", branch)
	}

	return fmt.Sprintf(setupSuccessMessageTmpl, sess.repo, channelTitle, monitorTypeDesc, branchInfo)
}
