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
