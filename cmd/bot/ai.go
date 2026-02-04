package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
)

var aiConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type chatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []messageParam `json:"messages"`
}

type messageParam struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// init
func initAI(apiKey, baseURL, model string) {
	aiConfig.APIKey = strings.TrimSpace(apiKey)
	aiConfig.BaseURL = strings.TrimSpace(baseURL)
	aiConfig.Model = strings.TrimSpace(model)

	if aiConfig.BaseURL == "" {
		aiConfig.BaseURL = "https://api.openai.com/v1"
	}
	// 兼容接口
	aiConfig.BaseURL = strings.TrimSuffix(aiConfig.BaseURL, "/")
	
	if aiConfig.Model == "" {
		aiConfig.Model = "gpt-5.2"
	}
}

// 检测文本是否主要为中文
func isChinese(text string) bool {
	chineseCount := 0
	totalCount := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			totalCount++
			if unicode.Is(unicode.Han, r) {
				chineseCount++
			}
		}
	}
	if totalCount == 0 {
		return false
	}
	return float64(chineseCount)/float64(totalCount) > 0.3
}

// 翻译文本
func translateText(text string) (string, error) {
	if aiConfig.APIKey == "" {
		return "", nil 
	}

	// 中文不翻译
	if isChinese(text) {
		return "", nil
	}

	reqBody := chatCompletionRequest{
		Model: aiConfig.Model,
		Messages: []messageParam{
		{
				Role:    "system",
				Content: "你是一个凉心开发的技术翻译助手。请将以下 GitHub Commit Message 翻译成简洁的中文。规则：1. 不要包含'翻译：'等前缀，直接输出译文；2. 保留 commit 类型前缀不翻译（如 feat、fix、refactor、chore、docs、style、test、perf 等，以及括号内的 scope），只翻译冒号后面的内容；3. 标题行和列表项之间用空行分隔，让格式更清晰。",
			},
			{
				Role:    "user",
				Content: text,
			},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := aiConfig.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+aiConfig.APIKey)

	// 10 Seconds timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(body))
	}

	var aiResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", err
	}

	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("empty choices from ai")
	}

	return aiResp.Choices[0].Message.Content, nil
}
