package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"simpleword/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

// settingsPage 负责单词本设置，并在选择删除后展示删除确认态。
type settingsPage struct {
	common
	deck       model.Deck
	stats      model.DeckStats
	cursor     int  // 0: 开始学习, 1: 删除单词本
	confirming bool // 展示删除确认框
	err        error
}

func newSettingsPage(c common, deck model.Deck) settingsPage {
	return settingsPage{common: c, deck: deck}
}

// statsLoadedMsg 表示单词本统计加载完成。
type statsLoadedMsg struct {
	stats model.DeckStats
	err   error
}

// deckDeletedMsg 表示删除操作完成。
type deckDeletedMsg struct{ err error }

func (p settingsPage) loadStatsCmd() tea.Cmd {
	s, deckID := p.store, p.deck.ID
	return func() tea.Msg {
		st, err := s.Stats(deckID)
		return statsLoadedMsg{stats: st, err: err}
	}
}

func (p settingsPage) deleteDeckCmd() tea.Cmd {
	s, deckID, mediaDir, deckName := p.store, p.deck.ID, p.mediaDir, p.deck.Name
	return func() tea.Msg {
		if err := s.DeleteDeck(deckID); err != nil {
			return deckDeletedMsg{err: err}
		}
		return deckDeletedMsg{err: os.RemoveAll(filepath.Join(mediaDir, deckName))}
	}
}

func (p settingsPage) Init() tea.Cmd {
	return p.loadStatsCmd()
}

func (p settingsPage) Update(msg tea.Msg) (page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		return p, nil

	case statsLoadedMsg:
		p.err = msg.err
		p.stats = msg.stats
		return p, nil

	case deckDeletedMsg:
		if msg.err != nil {
			return p, navigate(newDeckListPageWithStatus(p.common, "删除失败: "+msg.err.Error()))
		}
		return p, navigate(newDeckListPageWithStatus(p.common, "已删除单词本"))

	case tea.KeyMsg:
		if p.confirming {
			return p.handleConfirmKey(msg)
		}
		return p.handleKey(msg)
	}
	return p, nil
}

func (p settingsPage) handleKey(msg tea.KeyMsg) (page, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return p, navigate(newDeckListPage(p.common))
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "j":
		if p.cursor < 1 {
			p.cursor++
		}
	case "enter", " ":
		if p.cursor == 0 {
			return p, navigate(newStudyPage(p.common, p.deck))
		}
		p.confirming = true
	}
	return p, nil
}

func (p settingsPage) handleConfirmKey(msg tea.KeyMsg) (page, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return p, p.deleteDeckCmd()
	case "n", "esc", "q":
		p.confirming = false
	}
	return p, nil
}

func (p settingsPage) View() string {
	if p.confirming {
		return p.viewDeleteConfirm()
	}
	return p.viewSettings()
}

func (p settingsPage) viewSettings() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("⚙ 单词本设置 · " + p.deck.Name))
	b.WriteString("\n\n")

	st := p.stats
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
		if i == p.cursor {
			b.WriteString(selectedStyle.Render("▸ " + it))
		} else {
			b.WriteString(normalStyle.Render("  " + it))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n↑/↓ 选择 · enter 确认 · esc 返回"))
	return p.center(p.box().Render(b.String()))
}

func (p settingsPage) viewDeleteConfirm() string {
	var b strings.Builder
	b.WriteString(termStyle.Render("⚠ 确认删除"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render(fmt.Sprintf("确定要删除单词本「%s」吗？", p.deck.Name)))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("此操作不可恢复。"))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("y 确认删除 · n 取消"))
	return p.center(p.box().Render(b.String()))
}
