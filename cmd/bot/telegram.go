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
func (c *telegramClient) sendMessage(chatID int64, text, parseMode string, disablePreview bool, replyMarkup string) (*message, error) {
	Logger.Debug("💬 Sending message to %d (%d chars)", chatID, len(text))
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
