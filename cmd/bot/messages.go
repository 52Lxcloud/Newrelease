package main

import (
	"fmt"
)

// Messages 消息模板
// 所有消息都使用 MarkdownV2 格式
var Messages = struct {
	// 帮助和说明
	Help      func() string
	ListEmpty func() string

	// 错误消息
	ErrorFormat          func() string
	ErrorInvalidRepo     func() string
	ErrorRepoExists      func() string
	ErrorChannelNotFound func() string
	ErrorBotNotAdmin     func() string
	ErrorUnexpected      func() string
	ErrorDeleteFormat    func() string
	ErrorInvalidIndex    func() string

	// 成功消息
	SuccessAdded   func(repo, target, monitorType, branchInfo string) string
	SuccessDeleted func(repo string) string

	// 列表
	ListHeader func() string
	ListItem   func(index int, repo, branchInfo, monitorType, target string) string

	// 通知
	NotifyRelease func(repo, tag, body, translation, url string) string
	NotifyCommit  func(repoName, branch, message, translation, url string) string
}{
	// ============================================
	// 帮助消息
	// ============================================
	Help: func() string {
		return MDV2.JoinLines(
			MDV2.Bold("GitHub Release \\& Commit 监控机器人"),
			"",
			MDV2.Bold("可用命令："),
			"",
			MDV2.Nbsp("•", MDV2.CodeRaw("/list"), "\\-", "查看所有监控的仓库"),
			"",
			MDV2.Nbsp("•", MDV2.CodeRaw("/add"), "\\-", "添加仓库监控"),
			MDV2.Nbsp(" ", "格式：", MDV2.CodeRaw("/add owner/repo[:branch] [选项]")),
			"",
			"  选项：",
			MDV2.Nbsp(" ", MDV2.CodeRaw("-r"), ":", "监控 Release"),
			MDV2.Nbsp(" ", MDV2.CodeRaw("-c"), ":", "监控 Commit"),
			MDV2.Nbsp(" ", MDV2.CodeRaw("@channel"), ":", "发送到指定频道（默认私聊）"),
			"",
			"  示例：",
			MDV2.Nbsp(" ", MDV2.CodeRaw("/add nginx/nginx:master -r")),
			MDV2.Nbsp(" ", MDV2.CodeRaw("/add golang/go:dev -c")),
			MDV2.Nbsp(" ", MDV2.CodeRaw("/add facebook/react")),
			"",
			MDV2.Nbsp("•", MDV2.CodeRaw("/delete <序号>"), "\\-", "删除监控"),
			MDV2.Nbsp(" ", "示例：", MDV2.CodeRaw("/delete 1")),
			"",
			MDV2.Bold("提示："),
			"• 默认监控 Release 和 Commit",
			MDV2.Nbsp("•", "用", MDV2.CodeRaw(":branch"), "快速指定其他分支"),
			"• 频道需先添加机器人为管理员",
		)
	},

	// ============================================
	// 错误消息
	// ============================================
	ListEmpty: func() string {
		return MDV2.JoinLines(
			"📭 当前没有已添加的仓库。",
			"",
			MDV2.Nbsp("使用", MDV2.CodeRaw("/add owner/repo"), "添加监控"),
		)
	},

	ErrorFormat: func() string {
		return MDV2.JoinLines(
			"❌ 格式错误！",
			"",
			MDV2.Nbsp("使用方法：", MDV2.CodeRaw("/add owner/repo [选项]")),
			"",
			MDV2.Nbsp("发送", MDV2.CodeRaw("/start"), "查看详细帮助。"),
		)
	},

	ErrorInvalidRepo: func() string {
		return MDV2.JoinLines(
			MDV2.Nbsp("❌", MDV2.Bold("格式错误")),
			"",
			MDV2.Nbsp("请使用", MDV2.CodeRaw("owner/repository"), "格式"),
			MDV2.Nbsp("例如：", MDV2.CodeRaw("aiogram/aiogram")),
		)
	},

	ErrorRepoExists: func() string {
		return MDV2.JoinLines(
			MDV2.Nbsp("⚠️", MDV2.Bold("该仓库已存在相同配置")),
			"无需重复添加",
		)
	},

	ErrorChannelNotFound: func() string {
		return MDV2.JoinLines(
			MDV2.Nbsp("❌", MDV2.Bold("频道不存在")),
			"",
			"请检查用户名，并确保已添加机器人为管理员",
		)
	},

	ErrorBotNotAdmin: func() string {
		return MDV2.JoinLines(
			MDV2.Nbsp("⚠️", MDV2.Bold("权限不足")),
			"",
			"请将机器人添加为频道管理员",
			"并授予 \"发布消息\" 权限",
		)
	},

	ErrorUnexpected: func() string {
		return MDV2.JoinLines(
			MDV2.Nbsp("❌", MDV2.Bold("操作失败")),
			"",
			"发生未知错误，请稍后重试",
		)
	},

	ErrorDeleteFormat: func() string {
		return MDV2.JoinLines(
			"❌ 格式错误！",
			"",
			MDV2.Nbsp("使用方法：", MDV2.CodeRaw("/delete <序号>")),
			"",
			MDV2.Nbsp("先用", MDV2.CodeRaw("/list"), "查看序号。"),
		)
	},

	ErrorInvalidIndex: func() string {
		return "❌ 序号必须是大于 0 的数字！"
	},

	// ============================================
	// 成功消息
	// ============================================
	SuccessAdded: func(repo, target, monitorType, branchInfo string) string {
		lines := []string{
			MDV2.Bold("添加成功"),
			"",
			MDV2.Nbsp("📦", MDV2.Bold("仓库") + ":", MDV2.CodeRaw(repo)),
			MDV2.Nbsp("📢", MDV2.Bold("通知") + ":", target),
			MDV2.Nbsp("🔍", MDV2.Bold("监控") + ":", monitorType),
		}
		if branchInfo != "" {
			lines = append(lines, MDV2.Nbsp("🔀", MDV2.Bold("分支") + ":", MDV2.CodeRaw(branchInfo)))
		}
		lines = append(lines, "", "监控已启动，将在发现更新时通知你")
		return MDV2.JoinLines(lines...)
	},

	SuccessDeleted: func(repo string) string {
		return MDV2.JoinLines(
			MDV2.Nbsp("🗑️", MDV2.Bold("删除成功")),
			"",
			MDV2.Nbsp("已停止监控", MDV2.CodeRaw(repo)),
		)
	},

	// ============================================
	// 列表消息
	// ============================================
	ListHeader: func() string {
		return MDV2.Nbsp("📚", MDV2.Bold("已监控的仓库"))
	},

	ListItem: func(index int, repo, branchInfo, monitorType, target string) string {
		// 格式: *1\.* `owner/repo:branch`
		//       └─ 监控: Release + Commit
		//       └─ 通知: 私聊
		repoDisplay := repo
		if branchInfo != "" {
			repoDisplay = repo + ":" + branchInfo
		}
		return MDV2.JoinLines(
			fmt.Sprintf("*%d\\.* %s", index, MDV2.CodeRaw(repoDisplay)),
			fmt.Sprintf("└─ 监控: %s", monitorType),
			fmt.Sprintf("└─ 通知: %s", target),
		)
	},

	// ============================================
	// 通知消息
	// ============================================
	NotifyRelease: func(repo, tag, body, translation, url string) string {
		var lines []string

		// 标题
		lines = append(lines,
			MDV2.Nbsp("🎉", MDV2.Bold("new release")),
			"",
			"📦 "+MDV2.Escape(repo),
			"└─ "+MDV2.CodeRaw(tag),
		)

		// 翻译
		if translation != "" {
			lines = append(lines,
				"",
				MDV2.Bold("更新日志") + ":",
				MDV2.BlockquoteEscaped(translation),
			)
		}

		// 链接
		lines = append(lines,
			"",
			MDV2.LinkRaw("查看详情", url),
		)

		return MDV2.JoinLines(lines...)
	},

	NotifyCommit: func(repoName, branch, message, translation, url string) string {
		var lines []string

		// 标题
		lines = append(lines,
			MDV2.Nbsp("🔨", MDV2.Bold(fmt.Sprintf("new commits to %s:%s", MDV2.Escape(repoName), MDV2.Escape(branch)))),
			"",
			MDV2.CodeBlockRaw(message),
		)

		// 翻译（如果有）
		if translation != "" {
			lines = append(lines,
				"",
				MDV2.Bold("译") + ":",
				MDV2.BlockquoteEscaped(translation),
			)
		}

		// 链接
		lines = append(lines,
			"",
			MDV2.LinkRaw("查看详情", url),
		)

		return MDV2.JoinLines(lines...)
	},
}
