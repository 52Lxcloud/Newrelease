package main

import (
	"fmt"
	"strings"
)

// escapeMarkdown 转义 Markdown 特殊字符
func escapeMarkdown(text string) string {
	return markdownEscaper.Replace(text)
}

// parseCommand 解析命令
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

// buildRepoListMessage 构建仓库列表消息
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
	builder.WriteString("\n\n")
	for i, cfg := range configs {
		repo := escapeMarkdown(cfg.Repo)
		
		// 分支信息
		branchInfo := ""
		if cfg.MonitorCommit && cfg.Branch != "" && cfg.Branch != "main" {
			branchInfo = fmt.Sprintf(":%s", cfg.Branch)
		}
		
		// 通知目标
		target := "私聊"
		if cfg.ChannelID != 0 {
			channelTitle := strings.TrimSpace(cfg.ChannelTitle)
			if channelTitle != "" {
				target = escapeMarkdown(channelTitle)
			} else {
				target = fmt.Sprintf("频道 %d", cfg.ChannelID)
			}
		}

		// 监控类型
		var monitorType string
		if cfg.MonitorRelease && cfg.MonitorCommit {
			monitorType = "Release + Commit"
		} else if cfg.MonitorRelease {
			monitorType = "Release"
		} else if cfg.MonitorCommit {
			monitorType = "Commit"
		}

		// 多行格式
		builder.WriteString(fmt.Sprintf("*%d.* `%s%s`\n", i+1, repo, branchInfo))
		builder.WriteString(fmt.Sprintf("└─ 监控: %s\n", monitorType))
		builder.WriteString(fmt.Sprintf("└─ 通知: %s\n\n", target))
	}
	return strings.TrimSpace(builder.String()), nil
}
