package main

import (
	"fmt"
	"strings"
)

// MarkdownV2 格式化工具
// Telegram MarkdownV2 格式说明：
// - 必须转义的字符: _ * [ ] ( ) ~ ` > # + - = | { } . !
// - 在 code/pre 块中只需转义: ` \
// - 在链接 URL 中只需转义: ) \
var MDV2 = &mdv2{}

type mdv2 struct{}

// ============================================
// 转义函数
// ============================================

// escapeChars 需要在普通文本中转义的字符
var escapeChars = strings.NewReplacer(
	`\`, `\\`,
	`_`, `\_`,
	`*`, `\*`,
	`[`, `\[`,
	`]`, `\]`,
	`(`, `\(`,
	`)`, `\)`,
	`~`, `\~`,
	"`", "\\`",
	`>`, `\>`,
	`#`, `\#`,
	`+`, `\+`,
	`-`, `\-`,
	`=`, `\=`,
	`|`, `\|`,
	`{`, `\{`,
	`}`, `\}`,
	`.`, `\.`,
	`!`, `\!`,
)

// escapeCodeChars 代码块中需要转义的字符
var escapeCodeChars = strings.NewReplacer(
	`\`, `\\`,
	"`", "\\`",
)

// escapeURLChars URL 中需要转义的字符
var escapeURLChars = strings.NewReplacer(
	`\`, `\\`,
	`)`, `\)`,
)

// Escape 转义普通文本中的特殊字符
func (m *mdv2) Escape(text string) string {
	return escapeChars.Replace(text)
}

// EscapeCode 转义代码块中的字符
func (m *mdv2) EscapeCode(text string) string {
	return escapeCodeChars.Replace(text)
}

// EscapeURL 转义 URL 中的字符
func (m *mdv2) EscapeURL(url string) string {
	return escapeURLChars.Replace(url)
}

// ============================================
// 基础格式化
// ============================================

// Bold 粗体 *text*
func (m *mdv2) Bold(text string) string {
	return fmt.Sprintf("*%s*", text)
}

// Italic 斜体 _text_
func (m *mdv2) Italic(text string) string {
	return fmt.Sprintf("_%s_", text)
}

// Underline 下划线 __text__
func (m *mdv2) Underline(text string) string {
	return fmt.Sprintf("__%s__", text)
}

// Strikethrough 删除线 ~text~
func (m *mdv2) Strikethrough(text string) string {
	return fmt.Sprintf("~%s~", text)
}

// Spoiler 剧透/隐藏 ||text||
func (m *mdv2) Spoiler(text string) string {
	return fmt.Sprintf("||%s||", text)
}

// ============================================
// 代码格式化
// ============================================

// Code 内联代码 `text`
// 注意：代码内容会自动转义
func (m *mdv2) Code(text string) string {
	return fmt.Sprintf("`%s`", m.EscapeCode(text))
}

// CodeRaw 内联代码（不转义，用于已知安全的内容）
func (m *mdv2) CodeRaw(text string) string {
	return fmt.Sprintf("`%s`", text)
}

// CodeBlock 代码块
// ```
// text
// ```
func (m *mdv2) CodeBlock(text string) string {
	return fmt.Sprintf("```\n%s\n```", m.EscapeCode(text))
}

// CodeBlockRaw 代码块（不转义）
func (m *mdv2) CodeBlockRaw(text string) string {
	return fmt.Sprintf("```\n%s\n```", text)
}

// CodeBlockLang 带语言的代码块
// ```language
// text
// ```
func (m *mdv2) CodeBlockLang(lang, text string) string {
	return fmt.Sprintf("```%s\n%s\n```", lang, m.EscapeCode(text))
}

// ============================================
// 链接格式化
// ============================================

// Link 创建链接 [text](url)
// text 和 url 都会自动转义
func (m *mdv2) Link(text, url string) string {
	return fmt.Sprintf("[%s](%s)", m.Escape(text), m.EscapeURL(url))
}

// LinkRaw 创建链接（不转义，用于已知安全的内容）
func (m *mdv2) LinkRaw(text, url string) string {
	return fmt.Sprintf("[%s](%s)", text, url)
}

// Mention 用户提及 [name](tg://user?id=xxx)
func (m *mdv2) Mention(name string, userID int64) string {
	return fmt.Sprintf("[%s](tg://user?id=%d)", m.Escape(name), userID)
}

// ============================================
// 引用格式化 (MarkdownV2 特有)
// ============================================

// Blockquote 引用块
// 自动在每行开头添加 >
func (m *mdv2) Blockquote(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = ">" + line
	}
	return strings.Join(lines, "\n")
}

// BlockquoteEscaped 转义后的引用块
func (m *mdv2) BlockquoteEscaped(text string) string {
	return m.Blockquote(m.Escape(text))
}

// ExpandableQuote 可展开的引用块
// 格式: **>第一行
//       >其他行
//       >最后一行||
func (m *mdv2) ExpandableQuote(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}
	
	var result strings.Builder
	result.WriteString("**>")
	result.WriteString(lines[0])
	
	for i := 1; i < len(lines); i++ {
		result.WriteString("\n>")
		result.WriteString(lines[i])
	}
	result.WriteString("||")
	
	return result.String()
}

// ============================================
// 组合格式化
// ============================================

// BoldItalic 粗斜体 *_text_*
func (m *mdv2) BoldItalic(text string) string {
	return fmt.Sprintf("*_%s_*", text)
}

// BoldUnderline 粗体+下划线 *__text__*
func (m *mdv2) BoldUnderline(text string) string {
	return fmt.Sprintf("*__%s__*", text)
}

// ============================================
// 便捷方法
// ============================================

// Text 转义普通文本（Escape 的别名）
func (m *mdv2) Text(text string) string {
	return m.Escape(text)
}

// Line 创建带换行的文本
func (m *mdv2) Line(parts ...string) string {
	return strings.Join(parts, "") + "\n"
}

// Join 连接多个部分
func (m *mdv2) Join(parts ...string) string {
	return strings.Join(parts, "")
}

// JoinLines 用换行连接
func (m *mdv2) JoinLines(parts ...string) string {
	return strings.Join(parts, "\n")
}

// Nbsp 空格分隔
func (m *mdv2) Nbsp(parts ...string) string {
	return strings.Join(parts, " ")
}
