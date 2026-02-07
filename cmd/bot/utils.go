package main

import (
	"fmt"
	"strings"
)

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
		return Messages.ListEmpty(), nil
	}

	var builder strings.Builder
	builder.WriteString(Messages.ListHeader())
	builder.WriteString("\n\n")
	
	for i, cfg := range configs {
		// 分支信息（非 main 分支才显示）
		branchInfo := ""
		if cfg.MonitorCommit && cfg.Branch != "" && cfg.Branch != "main" {
			branchInfo = cfg.Branch
		}

		// 通知目标
		target := "私聊"
		if cfg.ChannelID != 0 {
			channelTitle := strings.TrimSpace(cfg.ChannelTitle)
			if channelTitle != "" {
				target = MDV2.Escape(channelTitle)
			} else {
				target = "群组"
			}
			// 如果有话题，显示 群组 > 话题名（话题名就是仓库名）
			if cfg.ThreadID > 0 {
				target = fmt.Sprintf("%s \\> %s", target, MDV2.Escape(cfg.RepoName))
			}
		}

		// 监控类型
		var monitorType string
		if cfg.MonitorRelease && cfg.MonitorCommit {
			monitorType = "Release \\+ Commit"
		} else if cfg.MonitorRelease {
			monitorType = "Release"
		} else if cfg.MonitorCommit {
			monitorType = "Commit"
		}

		// 构建列表项
		builder.WriteString(Messages.ListItem(i+1, MDV2.Escape(cfg.Repo), branchInfo, monitorType, target))
		builder.WriteString("\n\n")
	}
	
	return strings.TrimSpace(builder.String()), nil
}
