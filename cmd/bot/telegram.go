package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Telegram API 响应结构
type telegramResponse struct {
	Ok          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
}

// Telegram 消息相关结构
type update struct {
	UpdateID int      `json:"update_id"`
	Message  *message `json:"message"`
}

type message struct {
	MessageID int    `json:"message_id"`
	From      *user  `json:"from"`
	Chat      *chat  `json:"chat"`
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
	IsForum  bool   `json:"is_forum"` // 是否开启话题功能
}

type chatMember struct {
	User user `json:"user"`
}

// telegramClient Telegram 客户端
type telegramClient struct {
	baseURL    string
	httpClient *http.Client
	botID      int64
}

// newTelegramClient 创建新的 Telegram 客户端
func newTelegramClient(token string) *telegramClient {
	return &telegramClient{
		baseURL:    "https://api.telegram.org/bot" + token + "/",
		httpClient: &http.Client{Timeout: 65 * time.Second},
	}
}

// call 调用 Telegram API
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

// getMe 获取机器人信息
func (c *telegramClient) getMe() (*user, error) {
	var me user
	if err := c.call("getMe", nil, &me); err != nil {
		return nil, err
	}
	return &me, nil
}

// getUpdates 获取消息更新
func (c *telegramClient) getUpdates(offset int) ([]update, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	params.Set("timeout", "60")
	params.Set("allowed_updates", `["message"]`)

	var updates []update
	if err := c.call("getUpdates", params, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// sendMessage 发送消息
// threadID: 群组话题 ID，为 0 时不指定话题
func (c *telegramClient) sendMessage(chatID int64, text, parseMode string, disablePreview bool, replyMarkup string, threadID int64) (*message, error) {
	Logger.Debug("💬 Sending message to %d (topic: %d, %d chars)", chatID, threadID, len(text))
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
	if threadID > 0 {
		params.Set("message_thread_id", strconv.FormatInt(threadID, 10))
	}
	var msg message
	if err := c.call("sendMessage", params, &msg); err != nil {
		log.Printf("❌ Telegram sendMessage failed: %v", err)
		return nil, err
	}
	Logger.Debug("✅ Message sent (id: %d)", msg.MessageID)
	return &msg, nil
}

// getChat 获取频道/群聊信息
func (c *telegramClient) getChat(chatIDOrUsername string) (*chat, error) {
	params := url.Values{}
	params.Set("chat_id", chatIDOrUsername)
	var tgChat chat
	if err := c.call("getChat", params, &tgChat); err != nil {
		return nil, err
	}
	return &tgChat, nil
}

// getChatAdministrators 获取管理员列表
func (c *telegramClient) getChatAdministrators(chatID int64) ([]chatMember, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))
	var admins []chatMember
	if err := c.call("getChatAdministrators", params, &admins); err != nil {
		return nil, err
	}
	return admins, nil
}

// forumTopic 话题结构
type forumTopic struct {
	MessageThreadID   int64  `json:"message_thread_id"`
	Name              string `json:"name"`
	IconColor         int    `json:"icon_color"`
	IconCustomEmojiID string `json:"icon_custom_emoji_id,omitempty"`
}

// sticker 贴纸结构（用于获取话题图标）
type sticker struct {
	CustomEmojiID string `json:"custom_emoji_id"`
	Emoji         string `json:"emoji"`
}

// cachedTopicIconEmojis 缓存的话题图标 emoji
var cachedTopicIconEmojis []sticker

// getForumTopicIconStickers 获取可用的话题图标 emoji 列表
func (c *telegramClient) getForumTopicIconStickers() ([]sticker, error) {
	// 如果已缓存，直接返回
	if len(cachedTopicIconEmojis) > 0 {
		return cachedTopicIconEmojis, nil
	}
	
	var stickers []sticker
	if err := c.call("getForumTopicIconStickers", nil, &stickers); err != nil {
		return nil, err
	}
	
	// 缓存结果
	cachedTopicIconEmojis = stickers
	Logger.Debug("📦 Fetched %d forum topic icon stickers from Telegram", len(stickers))
	return stickers, nil
}

// createForumTopic 在群组中创建话题
// 返回创建的话题 ID，图标随机选择 emoji
func (c *telegramClient) createForumTopic(chatID int64, name string) (*forumTopic, error) {
	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))
	params.Set("name", name)
	
	// 获取可用的 emoji 图标并随机选择
	stickers, err := c.getForumTopicIconStickers()
	if err != nil {
		log.Printf("❌ Failed to get topic icon stickers: %v", err)
		return nil, err
	}
	
	if len(stickers) > 0 {
		idx := time.Now().UnixNano() % int64(len(stickers))
		emoji := stickers[idx]
		// 同时尝试两个可能的参数名，以防万一
		params.Set("icon_custom_emoji_id", emoji.CustomEmojiID)
		
		Logger.Debug("🎨 Choosing topic icon: %s (id: %s) for topic '%s'", emoji.Emoji, emoji.CustomEmojiID, name)
	} else {
		Logger.Debug("⚠️ No stickers returned from getForumTopicIconStickers")
	}
	
	var topic forumTopic
	if err := c.call("createForumTopic", params, &topic); err != nil {
		log.Printf("❌ Failed to create forum topic: %v", err)
		return nil, err
	}
	Logger.Debug("✅ Forum topic created (thread_id: %d)", topic.MessageThreadID)
	return &topic, nil
}

