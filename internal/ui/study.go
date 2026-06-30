package ui

import (
	"fmt"
	"strings"

	"simpleword/internal/audio"
	"simpleword/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// roundSize 是每一轮抽取的单词数。
const roundSize = 7

// studyPage 负责背单词，并在本轮全部掌握后展示祝贺态。
type studyPage struct {
	common
	deck      model.Deck
	round     []model.Word
	roundIdx  int
	showDef   bool
	defScroll int
	done      bool // 本轮全部掌握，展示祝贺框
	err       error
}

func newStudyPage(c common, deck model.Deck) studyPage {
	return studyPage{common: c, deck: deck}
}

// roundLoadedMsg 表示一轮单词加载完成。
type roundLoadedMsg struct {
	words []model.Word
	err   error
}

func (p studyPage) loadRoundCmd() tea.Cmd {
	s, deckID := p.store, p.deck.ID
	return func() tea.Msg {
		words, err := s.PickRound(deckID, roundSize)
		return roundLoadedMsg{words: words, err: err}
	}
}

// playCurrentCmd 播放当前单词语音（副作用命令）。
func (p studyPage) playCurrentCmd() tea.Cmd {
	if p.roundIdx >= len(p.round) {
		return nil
	}
	file := p.round[p.roundIdx].Audio
	return func() tea.Msg {
		audio.Play(file)
		return nil
	}
}

func (p studyPage) Init() tea.Cmd {
	return p.loadRoundCmd()
}

func (p studyPage) Update(msg tea.Msg) (page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		return p, nil

	case roundLoadedMsg:
		if msg.err != nil {
			return p, navigate(newDeckListPageWithStatus(p.common, "加载失败: "+msg.err.Error()))
		}
		if len(msg.words) == 0 {
			return p, navigate(newDeckListPageWithStatus(p.common, "该单词本暂无单词"))
		}
		p.round = msg.words
		p.roundIdx = 0
		p.showDef = false
		p.defScroll = 0
		p.done = false
		return p, p.playCurrentCmd()

	case tea.KeyMsg:
		if p.done {
			return p.handleCongratsKey(msg)
		}
		return p.handleStudyKey(msg)
	}
	return p, nil
}

func (p studyPage) handleStudyKey(msg tea.KeyMsg) (page, tea.Cmd) {
	if len(p.round) == 0 {
		return p, nil
	}

	switch msg.String() {
	case "esc", "q":
		return p, navigate(newDeckListPage(p.common))
	case "s": // 查看完整释义
		p.showDef = true
		p.defScroll = 0
	case "up", "k": // 释义向上滚动
		if p.showDef && p.defScroll > 0 {
			p.defScroll--
		}
	case "down", "j": // 释义向下滚动
		if p.showDef && p.defScroll < p.maxDefScroll() {
			p.defScroll++
		}
	case "a": // 上一个
		if p.roundIdx > 0 {
			p.roundIdx--
			p.showDef = false
			p.defScroll = 0
			return p, p.playCurrentCmd()
		}
	case "d": // 掌握，进入下一个
		cur := p.round[p.roundIdx]
		s := p.store
		cmd := func() tea.Msg { s.MarkMastered(cur.ID); return nil }
		p.round[p.roundIdx].Mastered = true
		return p.advance(cmd)
	case " ", "enter": // 仍需复习，进入下一个（不标记掌握）
		return p.advance(nil)
	}
	return p, nil
}

// advance 前进到下一个未掌握的单词；若本轮全部掌握则进入祝贺态。
func (p studyPage) advance(extra tea.Cmd) (page, tea.Cmd) {
	p.showDef = false
	p.defScroll = 0
	n := len(p.round)
	for i := 1; i <= n; i++ {
		idx := (p.roundIdx + i) % n
		if !p.round[idx].Mastered {
			p.roundIdx = idx
			return p, tea.Batch(extra, p.playCurrentCmd())
		}
	}
	// 全部掌握
	p.done = true
	return p, extra
}

func (p studyPage) handleCongratsKey(msg tea.KeyMsg) (page, tea.Cmd) {
	switch msg.String() {
	case "enter", "y", " ": // 继续学习：拉取下一轮
		p.done = false
		return p, p.loadRoundCmd()
	case "esc", "n", "q": // 返回主界面
		return p, navigate(newDeckListPage(p.common))
	}
	return p, nil
}

func (p studyPage) View() string {
	if p.done {
		return p.viewCongrats()
	}
	return p.viewStudy()
}

func (p studyPage) viewStudy() string {
	if len(p.round) == 0 {
		return p.center(p.box().Render(mutedStyle.Render("正在加载单词...")))
	}

	w := p.round[p.roundIdx]
	var b strings.Builder

	progress := fmt.Sprintf("第 %d/%d 个 · %s", p.roundIdx+1, len(p.round), p.deck.Name)
	b.WriteString(mutedStyle.Render(progress))
	b.WriteString("\n\n")

	b.WriteString(termStyle.Render(w.Term))
	b.WriteString("\n")

	if w.Phonetic != "" {
		b.WriteString(phoneticStyle.Render("/" + w.Phonetic + "/"))
		b.WriteString("\n")
	}

	if w.Audio != "" && audio.Available() {
		b.WriteString(mutedStyle.Render("🔊 自动播放中"))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if p.showDef {
		def := w.Definition
		if def == "" {
			def = "（无释义）"
		}
		b.WriteString(defStyle.Render(p.visibleDefinition(def)))
		if p.maxDefScroll() > 0 {
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render(p.defScrollIndicator()))
		}
	} else {
		b.WriteString(mutedStyle.Render("按 s 查看释义"))
	}

	help := "d 掌握 · s 释义 · ↑/↓ 滚动 · a 上一个 · 空格 跳过 · esc 返回"
	b.WriteString(helpStyle.Render("\n\n" + help))
	return p.center(p.box().Render(b.String()))
}

func (p studyPage) viewCongrats() string {
	var b strings.Builder
	b.WriteString(okStyle.Render("🎉 恭喜！本轮单词已全部掌握！"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("是否继续学习下一轮？"))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("enter 继续学习 · esc 返回主界面"))
	return p.center(congratsBox.Render(b.String()))
}

// ---- 释义滚动辅助 ----

func (p studyPage) defViewHeight() int {
	if p.height <= 0 {
		return 8
	}
	h := p.height - 13
	if h < 1 {
		return 1
	}
	return h
}

func (p studyPage) defViewWidth() int {
	if p.width <= 0 {
		return 80
	}
	// boxStyle 左右边框 2 + 左右 padding 4，留 2 列避免贴边。
	w := p.width - 8
	if w < 1 {
		return 1
	}
	return w
}

func (p studyPage) definitionLines(def string) []string {
	var lines []string
	for _, line := range strings.Split(def, "\n") {
		lines = append(lines, p.wrapLine(line)...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func (p studyPage) wrapLine(line string) []string {
	width := p.defViewWidth()
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

func (p studyPage) maxDefScroll() int {
	if p.roundIdx >= len(p.round) {
		return 0
	}
	def := p.round[p.roundIdx].Definition
	if def == "" {
		def = "（无释义）"
	}
	max := len(p.definitionLines(def)) - p.defViewHeight()
	if max < 0 {
		return 0
	}
	return max
}

func (p studyPage) visibleDefinition(def string) string {
	lines := p.definitionLines(def)
	start := p.defScroll
	if start > len(lines) {
		start = len(lines)
	}
	end := start + p.defViewHeight()
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

func (p studyPage) defScrollIndicator() string {
	max := p.maxDefScroll()
	if max == 0 {
		return ""
	}
	return fmt.Sprintf("%s %d/%d %s", scrollMark(p.defScroll > 0, "↑"), p.defScroll+1, max+1, scrollMark(p.defScroll < max, "↓"))
}

func scrollMark(show bool, mark string) string {
	if show {
		return mark
	}
	return " "
}
