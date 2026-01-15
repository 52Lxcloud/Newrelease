package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	configFile     = "/data/configs.json"
	checkInterval  = 60 * time.Second
	initialDelay   = 15 * time.Second
	repoCheckDelay = 2 * time.Second
	defaultBranch  = "main"
)

var repoRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+$`)
var markdownEscaper = strings.NewReplacer(
	"\\", "\\\\",
	"_", "\\_",
	"*", "\\*",
	"`", "\\`",
	"[", "\\[",
	"]", "\\]",
)

const (
	startMessage        = "你好，管理员！请选择一个操作："
	repoPromptMessage   = "请发送你要监控的仓库。\n格式为 `owner/repository`（例如：`aiogram/aiogram`）。"
	cancelMessage       = "已取消设置。你可以发送 /start 重新开始。"
	listEmptyMessage    = "当前没有已添加的仓库。"
	listHeaderMessage   = "已添加的仓库："
	invalidRepoMessage  = "格式不正确。\n请使用 `owner/repository` 的格式后重试。"
	monitorTypePromptMessage = "请选择要监控的类型："
	branchPromptMessage       = "请选择要监控的分支："
	branchCustomPromptMessage = "请输入要监控的分支名称："
	channelPromptMessage      = "请选择通知方式："
	channelCustomPromptMessage = "请发送频道的用户名（例如：`@yourchannel`）。"
	channelAcceptedMessage  = "好的！现在，请把本机器人添加为你的 Telegram 频道*管理员*并授予「发布消息」权限。\n\n完成后，请发送频道的用户名（例如：`@yourchannel`）。"
	channelNotFoundMessage = "找不到该频道。请检查用户名，并确保已添加机器人。"
	botNotAdminMessage     = "我还不是该频道的管理员。请确保机器人拥有“发布消息”权限后再试。"
	unexpectedErrorMessage = "发生了未知错误，请稍后再试。"
	setupSuccessMessageTmpl = "✅ 设置成功！\n\n*仓库*: `%s`\n*通知方式*: %s\n*监控类型*: %s\n%s"
	repoAcceptedMessageTmpl = "好的！我将监控 `%s`。\n\n现在，请把本机器人添加为你的 Telegram 频道*管理员*并授予“发布消息”权限。\n\n完成后，请发送频道的用户名（例如：`@yourchannel`）。"
	releaseMessageTmpl      = "*新版本发布：%s*\n\n*仓库*: `%s`\n*标签*: `%s`\n\n[在 GitHub 查看 Release](%s)"
	commitMessageTmpl       = "*新提交*\n\n*仓库*: `%s`\n*分支*: `%s`\n*作者*: %s\n*信息*: %s\n*提交*: `%s`\n\n[查看提交](%s)"
	telegramParseModeMarkdown = "Markdown"
)

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

type repoConfig struct {
	Repo           string  `json:"repo"`
	ChannelID      int64   `json:"channel_id,omitempty"`      // 0 或不填表示发给管理员
	ChannelTitle   string  `json:"channel_title,omitempty"`
	MonitorRelease bool    `json:"monitor_releases"`
	MonitorCommit  bool    `json:"monitor_commits"`
	Branch         string  `json:"branch,omitempty"`
	LastReleaseID  *int64  `json:"last_release_id"`
	LastCommitSHA  *string `json:"last_commit_sha"`
}

type gitHubRelease struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type gitCommit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Commit  struct {
		Author struct {
			Name string `json:"name"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
}

type telegramResponse struct {
	Ok          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
}

type update struct {
	UpdateID      int            `json:"update_id"`
	Message       *message       `json:"message"`
	CallbackQuery *callbackQuery `json:"callback_query"`
}

type message struct {
	MessageID int   `json:"message_id"`
	From      *user `json:"from"`
	Chat      *chat `json:"chat"`
	Text      string `json:"text"`
}

type user struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type chat struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Username string `json:"username"`
	Type     string `json:"type"`
}

type chatMember struct {
	User user `json:"user"`
}

type callbackQuery struct {
	ID      string   `json:"id"`
	From    *user    `json:"from"`
	Message *message `json:"message"`
	Data    string   `json:"data"`
}

type telegramClient struct {
	baseURL    string
	httpClient *http.Client
	botID      int64
}

func newTelegramClient(token string) *telegramClient {
	return &telegramClient{
		baseURL:    "https://api.telegram.org/bot" + token + "/",
		httpClient: &http.Client{Timeout: 65 * time.Second},
	}
}

func (c *telegramClient) call(method string, params url.Values, result interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	req, err := http.NewRequest("POST", c.baseURL+method, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var apiResp telegramResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return err
	}
	if !apiResp.Ok {
		return fmt.Errorf("telegram api error %d: %s", apiResp.ErrorCode, apiResp.Description)
	}
	if result != nil {
		if err := json.Unmarshal(apiResp.Result, result); err != nil {
			return err
		}
	}
	return nil
}

func (c *telegramClient) getMe() (*user, error) {
	var me user
	if err := c.call("getMe", nil, &me); err != nil {
		return nil, err
	}
	return &me, nil
}

func (c *telegramClient) getUpdates(offset int) ([]update, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	params.Set("timeout", "60")
	params.Set("allowed_updates", `["message","callback_query"]`)

	var updates []update
	if err := c.call("getUpdates", params, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func (c *telegramClient) sendMessage(chatID int64, text, parseMode string, disablePreview bool, replyMarkup string) (*message, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))
	params.Set("text", text)
	if parseMode != "" {
		params.Set("parse_mode", parseMode)
	}
	if disablePreview {
		params.Set("disable_web_page_preview", "true")
	}
	if replyMarkup != "" {
		params.Set("reply_markup", replyMarkup)
	}
	var msg message
	if err := c.call("sendMessage", params, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *telegramClient) editMessageText(chatID int64, messageID int, text, parseMode string, disablePreview bool, replyMarkup string) error {
	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))
	params.Set("message_id", strconv.Itoa(messageID))
	params.Set("text", text)
	if parseMode != "" {
		params.Set("parse_mode", parseMode)
	}
	if disablePreview {
		params.Set("disable_web_page_preview", "true")
	}
	if replyMarkup != "" {
		params.Set("reply_markup", replyMarkup)
	}
	return c.call("editMessageText", params, nil)
}

func (c *telegramClient) deleteMessage(chatID int64, messageID int) error {
	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))
	params.Set("message_id", strconv.Itoa(messageID))
	return c.call("deleteMessage", params, nil)
}

func (c *telegramClient) getChat(chatIDOrUsername string) (*chat, error) {
	params := url.Values{}
	params.Set("chat_id", chatIDOrUsername)
	var tgChat chat
	if err := c.call("getChat", params, &tgChat); err != nil {
		return nil, err
	}
	return &tgChat, nil
}

func (c *telegramClient) getChatAdministrators(chatID int64) ([]chatMember, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))
	var admins []chatMember
	if err := c.call("getChatAdministrators", params, &admins); err != nil {
		return nil, err
	}
	return admins, nil
}

func (c *telegramClient) answerCallbackQuery(callbackID, text string, showAlert bool) error {
	params := url.Values{}
	params.Set("callback_query_id", callbackID)
	if text != "" {
		params.Set("text", text)
	}
	if showAlert {
		params.Set("show_alert", "true")
	}
	return c.call("answerCallbackQuery", params, nil)
}

type setupState int

const (
	stateIdle setupState = iota
	stateWaitingRepo
	stateWaitingMonitorType
	stateWaitingBranch
	stateWaitingBranchCustom
	stateWaitingChannelType
	stateWaitingChannel
)

type setupSession struct {
	state          setupState
	repo           string
	monitorRelease bool
	monitorCommit  bool
	branch         string
	lastBotMsgID   int
	chatID         int64
}

var (
	sessionMu sync.Mutex
	session   = setupSession{state: stateIdle}
	configMu  sync.Mutex
)

func setSession(s setupSession) {
	sessionMu.Lock()
	session = s
	sessionMu.Unlock()
}

func getSession() setupSession {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	return session
}

func loadConfigs() ([]repoConfig, error) {
	configMu.Lock()
	defer configMu.Unlock()

	_, err := os.Stat(configFile)
	if errors.Is(err, os.ErrNotExist) {
		return []repoConfig{}, nil
	}
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []repoConfig{}, nil
	}

	var configs []repoConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return []repoConfig{}, nil
	}
	return configs, nil
}

func saveConfigs(configs []repoConfig) error {
	configMu.Lock()
	defer configMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(configs, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0o644)
}

func parseCommand(text string) string {
	if !strings.HasPrefix(text, "/") {
		return ""
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	cmd := fields[0]
	if at := strings.Index(cmd, "@"); at != -1 {
		cmd = cmd[:at]
	}
	return cmd
}

func escapeMarkdown(text string) string {
	return markdownEscaper.Replace(text)
}

func startKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"添加仓库","callback_data":"%s"}],[{"text":"查看已添加仓库","callback_data":"%s"}],[{"text":"取消","callback_data":"%s"}]]}`, callbackAddRepo, callbackListRepos, callbackCancel)
}

func cancelKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"取消","callback_data":"%s"}]]}`, callbackCancel)
}

func monitorTypeKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"Release","callback_data":"%s"},{"text":"Commit","callback_data":"%s"}],[{"text":"Release+Commit","callback_data":"%s"}],[{"text":"取消","callback_data":"%s"}]]}`, 
		callbackMonitorRelease, callbackMonitorCommit, callbackMonitorBoth, callbackCancel)
}

func branchKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"main","callback_data":"%s"},{"text":"master","callback_data":"%s"}],[{"text":"自定义分支","callback_data":"%s"}],[{"text":"取消","callback_data":"%s"}]]}`,
		callbackBranchMain, callbackBranchMaster, callbackBranchCustom, callbackCancel)
}

func channelKeyboard() string {
	return fmt.Sprintf(`{"inline_keyboard":[[{"text":"私聊通知","callback_data":"%s"}],[{"text":"频道/群聊通知","callback_data":"%s"}],[{"text":"取消","callback_data":"%s"}]]}`,
		callbackChannelPrivate, callbackChannelCustom, callbackCancel)
}

func buildRepoListMessage() (string, error) {
	configs, err := loadConfigs()
	if err != nil {
		return "", err
	}
	if len(configs) == 0 {
		return listEmptyMessage, nil
	}

	var builder strings.Builder
	builder.WriteString(listHeaderMessage)
	builder.WriteString("\n")
	for i, cfg := range configs {
		repo := escapeMarkdown(cfg.Repo)
		channelTitle := strings.TrimSpace(cfg.ChannelTitle)
		if channelTitle == "" {
			channelTitle = fmt.Sprintf("频道ID %d", cfg.ChannelID)
		}
		channelTitle = escapeMarkdown(channelTitle)
		
		var monitorType string
		if cfg.MonitorRelease && cfg.MonitorCommit {
			monitorType = "Release + Commit"
		} else if cfg.MonitorRelease {
			monitorType = "Release"
		} else if cfg.MonitorCommit {
			monitorType = "Commit"
		} else {
			monitorType = "未设置"
		}
		
		branchInfo := ""
		if cfg.MonitorCommit && cfg.Branch != "" {
			branchInfo = fmt.Sprintf(" \\[%s\\]", cfg.Branch)
		}
		
		builder.WriteString(fmt.Sprintf("%d. `%s` -> %s (%s%s)\n", i+1, repo, channelTitle, monitorType, branchInfo))
	}
	return strings.TrimSpace(builder.String()), nil
}

func scheduledChecker(tg *telegramClient, adminID int64) {
	time.Sleep(initialDelay)
	client := &http.Client{Timeout: 10 * time.Second}

	for {
		log.Printf("Running scheduled check for new releases...")
		configs, err := loadConfigs()
		if err != nil {
			log.Printf("Failed to load configs: %v", err)
		} else if len(configs) == 0 {
			log.Printf("No configurations found. Skipping check.")
		} else {
			for i := range configs {
				if configs[i].MonitorRelease {
					release, err := getLatestRelease(client, configs[i].Repo)
					if err != nil {
						log.Printf("Error fetching GitHub release for %s: %v", configs[i].Repo, err)
					} else if release != nil {
						if configs[i].LastReleaseID == nil || *configs[i].LastReleaseID != release.ID {
							log.Printf("New release found for %s: %s", configs[i].Repo, release.Name)
							if configs[i].LastReleaseID != nil {
							name := release.Name
							if name == "" {
								name = release.TagName
							}
							name = escapeMarkdown(name)
							messageText := fmt.Sprintf(
								releaseMessageTmpl,
								name,
								configs[i].Repo,
								release.TagName,
								release.HTMLURL,
							)
							targetID := configs[i].ChannelID
							if targetID == 0 {
								targetID = adminID
							}
							if _, err := tg.sendMessage(targetID, messageText, telegramParseModeMarkdown, true, ""); err != nil {
								log.Printf("Failed to send message to %d: %v", targetID, err)
							}
						}

						latestID := release.ID
						configs[i].LastReleaseID = &latestID
						if err := saveConfigs(configs); err != nil {
							log.Printf("Failed to save configs: %v", err)
						}
					}
					}
				}

				if configs[i].MonitorCommit {
					branch := configs[i].Branch
					if branch == "" {
						branch = defaultBranch
					}
					commit, err := getLatestCommit(client, configs[i].Repo, branch)
					if err != nil {
						log.Printf("Error fetching GitHub commit for %s (branch: %s): %v", configs[i].Repo, branch, err)
					} else if commit != nil {
						if configs[i].LastCommitSHA == nil || *configs[i].LastCommitSHA != commit.SHA {
							if configs[i].LastCommitSHA != nil {
							subject := strings.TrimSpace(commit.Commit.Message)
							if subject == "" {
								subject = commit.SHA
							}
							subject = strings.SplitN(subject, "\n", 2)[0]
							subject = escapeMarkdown(subject)

							author := strings.TrimSpace(commit.Commit.Author.Name)
							if author == "" && commit.Author != nil {
								author = commit.Author.Login
							}
							if author == "" {
								author = "未知作者"
							}
							author = escapeMarkdown(author)

							shortSHA := commit.SHA
							if len(shortSHA) > 7 {
								shortSHA = shortSHA[:7]
							}

							messageText := fmt.Sprintf(
								commitMessageTmpl,
								configs[i].Repo,
								branch,
								author,
								subject,
								shortSHA,
								commit.HTMLURL,
							)
							targetID := configs[i].ChannelID
							if targetID == 0 {
								targetID = adminID
							}
							if _, err := tg.sendMessage(targetID, messageText, telegramParseModeMarkdown, true, ""); err != nil {
								log.Printf("Failed to send commit message to %d: %v", targetID, err)
							}
						}

						latestSHA := commit.SHA
						configs[i].LastCommitSHA = &latestSHA
						if err := saveConfigs(configs); err != nil {
							log.Printf("Failed to save configs: %v", err)
						}
					}
					}
				}

				time.Sleep(repoCheckDelay)
			}
		}

		log.Printf("Check finished. Waiting for %s.", checkInterval)
		time.Sleep(checkInterval)
	}
}

func getLatestRelease(client *http.Client, repo string) (*gitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "newrelease")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("Repository %s not found (404). It might be private or have a typo.", repo)
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d from GitHub", resp.StatusCode)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func getLatestCommit(client *http.Client, repo, branch string) (*gitCommit, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/commits?sha=%s&per_page=1", repo, url.QueryEscape(branch))
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "newrelease")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("Repository %s or branch %s not found (404).", repo, branch)
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d from GitHub", resp.StatusCode)
	}

	var commits []gitCommit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, nil
	}
	return &commits[0], nil
}

func main() {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		log.Fatal("FATAL: TELEGRAM_BOT_TOKEN is not set in the environment.")
	}

	adminIDRaw := strings.TrimSpace(os.Getenv("ADMIN_ID"))
	adminID, err := strconv.ParseInt(adminIDRaw, 10, 64)
	if err != nil {
		log.Fatal("FATAL: ADMIN_ID is not a valid integer or is not set in the environment.")
	}

	tg := newTelegramClient(token)
	me, err := tg.getMe()
	if err != nil {
		log.Fatalf("Failed to fetch bot info: %v", err)
	}
	tg.botID = me.ID

	log.Printf("Bot starting... Authorized Admin User ID is %d", adminID)
	go scheduledChecker(tg, adminID)

	offset := 0
	for {
		updates, err := tg.getUpdates(offset)
		if err != nil {
			log.Printf("Failed to fetch updates: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, upd := range updates {
			if upd.UpdateID >= offset {
				offset = upd.UpdateID + 1
			}
			if upd.CallbackQuery != nil {
				cb := upd.CallbackQuery
				if cb.From == nil {
					continue
				}
				if cb.From.ID != adminID {
					log.Printf("Unauthorized access attempt by user %d (%s %s)", cb.From.ID, cb.From.FirstName, cb.From.LastName)
					_ = tg.answerCallbackQuery(cb.ID, "没有权限", false)
					continue
				}

				chatID := int64(0)
				messageID := 0
				if cb.Message != nil && cb.Message.Chat != nil {
					chatID = cb.Message.Chat.ID
					messageID = cb.Message.MessageID
				}

				switch cb.Data {
				case callbackAddRepo:
					setSession(setupSession{state: stateWaitingRepo, lastBotMsgID: messageID, chatID: chatID})
					if chatID != 0 && messageID != 0 {
						if err := tg.editMessageText(chatID, messageID, repoPromptMessage, telegramParseModeMarkdown, false, cancelKeyboard()); err != nil {
							log.Printf("Failed to edit repo prompt message: %v", err)
						}
					}
					_ = tg.answerCallbackQuery(cb.ID, "", false)
				case callbackMonitorRelease:
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
				case callbackMonitorCommit:
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
				case callbackMonitorBoth:
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
				case callbackBranchMain:
					sess := getSession()
					if sess.state == stateWaitingBranch {
						sess.state = stateWaitingChannelType
						sess.branch = "main"
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
				case callbackBranchMaster:
					sess := getSession()
					if sess.state == stateWaitingBranch {
						sess.state = stateWaitingChannelType
						sess.branch = "master"
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
				case callbackBranchCustom:
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
				case callbackChannelPrivate:
					sess := getSession()
					if sess.state == stateWaitingChannelType {
						// 直接完成设置，发送给管理员
						configs, err := loadConfigs()
						if err != nil {
							log.Printf("Failed to load configs: %v", err)
							if chatID != 0 && messageID != 0 {
								if err := tg.editMessageText(chatID, messageID, unexpectedErrorMessage, "", false, startKeyboard()); err != nil {
									log.Printf("Failed to edit error message: %v", err)
								}
							}
							_ = tg.answerCallbackQuery(cb.ID, "", false)
							continue
						}

						newConfig := repoConfig{
							Repo:           sess.repo,
							ChannelID:      0, // 0 表示发给管理员
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
							continue
						}

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

						successMessage := fmt.Sprintf(setupSuccessMessageTmpl, sess.repo, "私聊", monitorTypeDesc, branchInfo)
						if chatID != 0 && messageID != 0 {
							if err := tg.editMessageText(chatID, messageID, successMessage, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
								log.Printf("Failed to edit success message: %v", err)
							}
						}
						setSession(setupSession{state: stateIdle})
						_ = tg.answerCallbackQuery(cb.ID, "", false)
					}
				case callbackChannelCustom:
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
				case callbackListRepos:
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
				case callbackCancel:
					setSession(setupSession{state: stateIdle})
					if chatID != 0 && messageID != 0 {
						if err := tg.editMessageText(chatID, messageID, cancelMessage, "", false, startKeyboard()); err != nil {
							log.Printf("Failed to edit cancel message: %v", err)
						}
					}
					_ = tg.answerCallbackQuery(cb.ID, "", false)
				default:
					_ = tg.answerCallbackQuery(cb.ID, "", false)
				}
				continue
			}
			if upd.Message == nil || upd.Message.From == nil || upd.Message.Chat == nil {
				continue
			}

			fromID := upd.Message.From.ID
			if fromID != adminID {
				log.Printf("Unauthorized access attempt by user %d (%s %s)", fromID, upd.Message.From.FirstName, upd.Message.From.LastName)
				continue
			}

			text := strings.TrimSpace(upd.Message.Text)
			if text == "" {
				continue
			}

			switch parseCommand(text) {
			case "/start":
				setSession(setupSession{state: stateIdle})
				if _, err := tg.sendMessage(upd.Message.Chat.ID, startMessage, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
					log.Printf("Failed to send start message: %v", err)
				}
				continue
			case "/list":
				messageText, err := buildRepoListMessage()
				if err != nil {
					log.Printf("Failed to build repo list: %v", err)
					if _, err := tg.sendMessage(upd.Message.Chat.ID, unexpectedErrorMessage, "", false, startKeyboard()); err != nil {
						log.Printf("Failed to send repo list error message: %v", err)
					}
				} else {
					if _, err := tg.sendMessage(upd.Message.Chat.ID, messageText, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
						log.Printf("Failed to send repo list message: %v", err)
					}
				}
				continue
			case "/cancel":
				setSession(setupSession{state: stateIdle})
				if _, err := tg.sendMessage(upd.Message.Chat.ID, cancelMessage, "", false, startKeyboard()); err != nil {
					log.Printf("Failed to send cancel message: %v", err)
				}
				continue
			}

			sess := getSession()
			switch sess.state {
			case stateWaitingRepo:
				// 删除之前的提示消息
				if sess.lastBotMsgID != 0 && sess.chatID != 0 {
					_ = tg.deleteMessage(sess.chatID, sess.lastBotMsgID)
				}
				
				if !repoRegexp.MatchString(text) {
					// 删除用户发送的错误格式消息
					_ = tg.deleteMessage(upd.Message.Chat.ID, upd.Message.MessageID)
					msg, err := tg.sendMessage(upd.Message.Chat.ID, invalidRepoMessage, telegramParseModeMarkdown, false, cancelKeyboard())
					if err != nil {
						log.Printf("Failed to send invalid repo message: %v", err)
					} else {
						sess.lastBotMsgID = msg.MessageID
						sess.chatID = upd.Message.Chat.ID
						setSession(sess)
					}
					continue
				}

				sess.state = stateWaitingMonitorType
				sess.repo = text
				setSession(sess)
				if _, err := tg.sendMessage(upd.Message.Chat.ID, monitorTypePromptMessage, telegramParseModeMarkdown, false, monitorTypeKeyboard()); err != nil {
					log.Printf("Failed to send monitor type prompt message: %v", err)
				}
			case stateWaitingBranchCustom:
				branch := strings.TrimSpace(text)
				if branch == "" {
					branch = defaultBranch
				}
				sess.state = stateWaitingChannelType
				sess.branch = branch
				setSession(sess)
				if _, err := tg.sendMessage(upd.Message.Chat.ID, channelPromptMessage, telegramParseModeMarkdown, false, channelKeyboard()); err != nil {
					log.Printf("Failed to send channel prompt message: %v", err)
				}
			case stateWaitingChannel:
				// 删除之前的提示消息
				if sess.lastBotMsgID != 0 && sess.chatID != 0 {
					_ = tg.deleteMessage(sess.chatID, sess.lastBotMsgID)
				}
				
				channelName := text
				tgChat, err := tg.getChat(channelName)
				if err != nil {
					msg, err := tg.sendMessage(upd.Message.Chat.ID, channelNotFoundMessage, "", false, cancelKeyboard())
					if err != nil {
						log.Printf("Failed to send channel not found message: %v", err)
					} else {
						sess.lastBotMsgID = msg.MessageID
						sess.chatID = upd.Message.Chat.ID
						setSession(sess)
					}
					continue
				}

				admins, err := tg.getChatAdministrators(tgChat.ID)
				if err != nil {
					log.Printf("Error validating channel %s: %v", channelName, err)
					msg, err := tg.sendMessage(upd.Message.Chat.ID, unexpectedErrorMessage, "", false, cancelKeyboard())
					if err != nil {
						log.Printf("Failed to send unexpected error message: %v", err)
					} else {
						sess.lastBotMsgID = msg.MessageID
						sess.chatID = upd.Message.Chat.ID
						setSession(sess)
					}
					continue
				}

				isBotAdmin := false
				for _, admin := range admins {
					if admin.User.ID == tg.botID {
						isBotAdmin = true
						break
					}
				}
				if !isBotAdmin {
					msg, err := tg.sendMessage(upd.Message.Chat.ID, botNotAdminMessage, "", false, cancelKeyboard())
					if err != nil {
						log.Printf("Failed to send bot not admin message: %v", err)
					} else {
						sess.lastBotMsgID = msg.MessageID
						sess.chatID = upd.Message.Chat.ID
						setSession(sess)
					}
					continue
				}

				configs, err := loadConfigs()
				if err != nil {
					log.Printf("Failed to load configs: %v", err)
					if _, err := tg.sendMessage(upd.Message.Chat.ID, unexpectedErrorMessage, "", false, cancelKeyboard()); err != nil {
						log.Printf("Failed to send unexpected error message: %v", err)
					}
					continue
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
					if _, err := tg.sendMessage(upd.Message.Chat.ID, unexpectedErrorMessage, "", false, cancelKeyboard()); err != nil {
						log.Printf("Failed to send unexpected error message: %v", err)
					}
					continue
				}

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

				successMessage := fmt.Sprintf(setupSuccessMessageTmpl, sess.repo, escapeMarkdown(tgChat.Title), monitorTypeDesc, branchInfo)
				if _, err := tg.sendMessage(upd.Message.Chat.ID, successMessage, telegramParseModeMarkdown, false, startKeyboard()); err != nil {
					log.Printf("Failed to send success message: %v", err)
				}
				setSession(setupSession{state: stateIdle})
			default:
			}
		}
	}
}
