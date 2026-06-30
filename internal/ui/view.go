package ui

import (
	"fmt"
	"strings"

	"simpleword/internal/audio"

	"github.com/charmbracelet/lipgloss"
)

// View 实现 tea.Model。
func (m Model) View() string {
	var body string
	switch m.screen {
	case screenDeckList:
		body = m.viewDeckList()
	case screenImport:
		body = m.viewImport()
	case screenStudy:
		body = m.viewStudy()
	case screenSettings:
		body = m.viewSettings()
	case screenCongrats:
		body = m.viewCongrats()
	case screenDeleteConfirm:
		body = m.viewDeleteConfirm()
	}
	return m.center(body)
}

// center 在终端中居中显示内容。
func (m Model) center(body string) string {
	if m.width == 0 || m.height == 0 {
		return body
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

// box 返回一个自适应终端最大宽高的边框样式。
func (m Model) box() lipgloss.Style {
	s := boxStyle
	if m.width > 0 {
		// 减去边框占用的宽度
		if w := m.width - 2; w > 0 {
			s = s.Width(w)
		}
	}
	if m.height > 0 {
		if h := m.height - 2; h > 0 {
			s = s.Height(h)
		}
	}
	return s
}

func (m Model) viewDeckList() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📚 我的单词本"))
	b.WriteString("\n\n")

	if len(m.decks) == 0 {
		b.WriteString(mutedStyle.Render("还没有单词本，选择下方 + 导入一个吧"))
		b.WriteString("\n\n")
	}

	for i, d := range m.decks {
		line := fmt.Sprintf("  %s", d.Name)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("▸ " + d.Name))
		} else {
			b.WriteString(normalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// "+导入" 项
	plus := "+ 导入单词本"
	if m.cursor == len(m.decks) {
		b.WriteString(selectedStyle.Render("▸ " + plus))
	} else {
		b.WriteString(normalStyle.Render("  " + plus))
	}
	b.WriteString("\n")

	if m.status != "" {
		b.WriteString("\n" + okStyle.Render(m.status))
	}

	help := "↑/↓ 选择 · 空格 背单词 · enter 设置/导入 · q 退出"
	b.WriteString(helpStyle.Render("\n" + help))
	return m.box().Render(b.String())
}

func (m Model) viewImport() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📥 导入 Anki 单词本"))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("\nenter 确认导入 · esc 取消"))
	return m.box().Render(b.String())
}

func (m Model) viewStudy() string {
	w := m.round[m.roundIdx]
	var b strings.Builder

	// 进度
	progress := fmt.Sprintf("第 %d/%d 个 · %s", m.roundIdx+1, len(m.round), m.studyDeck.Name)
	b.WriteString(mutedStyle.Render(progress))
	b.WriteString("\n\n")

	// 单词
	b.WriteString(termStyle.Render(w.Term))
	b.WriteString("\n")

	// 音标
	if w.Phonetic != "" {
		b.WriteString(phoneticStyle.Render("/" + w.Phonetic + "/"))
		b.WriteString("\n")
	}

	// 语音提示
	if w.Audio != "" && audio.Available() {
		b.WriteString(mutedStyle.Render("🔊 自动播放中"))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// 释义
	if m.showDef {
		def := w.Definition
		if def == "" {
			def = "（无释义）"
		}
		b.WriteString(defStyle.Render(m.visibleDefinition(def)))
		if m.maxDefScroll() > 0 {
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render(m.defScrollIndicator()))
		}
	} else {
		b.WriteString(mutedStyle.Render("按 s 查看释义"))
	}

	help := "d 掌握 · s 释义 · ↑/↓ 滚动 · a 上一个 · 空格 跳过 · esc 返回"
	b.WriteString(helpStyle.Render("\n\n" + help))
	return m.box().Render(b.String())
}

func (m Model) defViewHeight() int {
	if m.height <= 0 {
		return 8
	}
	h := m.height - 13
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) definitionLines(def string) []string {
	var lines []string
	for _, line := range strings.Split(def, "\n") {
		lines = append(lines, m.wrapLine(line)...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func (m Model) wrapLine(line string) []string {
	width := m.defViewWidth()
	if width <= 0 || lipgloss.Width(line) <= width {
		return []string{line}
	}

	var lines []string
	var b strings.Builder
	for _, r := range line {
		if b.Len() > 0 && lipgloss.Width(b.String()+string(r)) > width {
			lines = append(lines, b.String())
			b.Reset()
		}
		b.WriteRune(r)
	}
	lines = append(lines, b.String())
	return lines
}

func (m Model) defViewWidth() int {
	if m.width <= 0 {
		return 80
	}
	// boxStyle 左右边框 2 + 左右 padding 4，留 2 列避免贴边。
	w := m.width - 8
	if w < 1 {
		return 1
	}
	return w
}

func (m Model) maxDefScroll() int {
	if m.roundIdx >= len(m.round) {
		return 0
	}
	def := m.round[m.roundIdx].Definition
	if def == "" {
		def = "（无释义）"
	}
	max := len(m.definitionLines(def)) - m.defViewHeight()
	if max < 0 {
		return 0
	}
	return max
}

func (m Model) visibleDefinition(def string) string {
	lines := m.definitionLines(def)
	start := m.defScroll
	if start > len(lines) {
		start = len(lines)
	}
	end := start + m.defViewHeight()
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

func (m Model) defScrollIndicator() string {
	max := m.maxDefScroll()
	if max == 0 {
		return ""
	}
	return fmt.Sprintf("%s %d/%d %s", scrollMark(m.defScroll > 0, "↑"), m.defScroll+1, max+1, scrollMark(m.defScroll < max, "↓"))
}

func scrollMark(show bool, mark string) string {
	if show {
		return mark
	}
	return " "
}

func (m Model) viewSettings() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("⚙ 单词本设置 · " + m.settingsDeck.Name))
	b.WriteString("\n\n")

	st := m.settingsStats
	pct := 0
	if st.Total > 0 {
		pct = st.Mastered * 100 / st.Total
	}
	b.WriteString(normalStyle.Render(fmt.Sprintf("单词总数：%d", st.Total)))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render(fmt.Sprintf("已掌握：%d", st.Mastered)))
	b.WriteString("\n")
	b.WriteString(okStyle.Render(fmt.Sprintf("学习进度：%d%%", pct)))
	b.WriteString("\n\n")

	items := []string{"▶ 开始学习", "🗑 删除单词本"}
	for i, it := range items {
		if i == m.settingsCursor {
			b.WriteString(selectedStyle.Render("▸ " + it))
		} else {
			b.WriteString(normalStyle.Render("  " + it))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n↑/↓ 选择 · enter 确认 · esc 返回"))
	return m.box().Render(b.String())
}

func (m Model) viewCongrats() string {
	var b strings.Builder
	b.WriteString(okStyle.Render("🎉 恭喜！本轮单词已全部掌握！"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("是否继续学习下一轮？"))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("enter 继续学习 · esc 返回主界面"))
	return congratsBox.Render(b.String())
}

func (m Model) viewDeleteConfirm() string {
	var b strings.Builder
	b.WriteString(termStyle.Render("⚠ 确认删除"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render(fmt.Sprintf("确定要删除单词本「%s」吗？", m.settingsDeck.Name)))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("此操作不可恢复。"))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("y 确认删除 · n 取消"))
	return m.box().Render(b.String())
}
