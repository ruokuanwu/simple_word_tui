package ui

import (
	"strings"

	"simpleword/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

// deckListPage 是主界面：单词本列表 + "+导入"项。
type deckListPage struct {
	common
	decks  []model.Deck
	cursor int // 0..len(decks)-1 为单词本，len(decks) 为 "+导入"
	status string
	err    error
}

func newDeckListPage(c common) deckListPage {
	return deckListPage{common: c}
}

// newDeckListPageWithStatus 构造一个带状态提示的列表页（用于导入/删除后回到主界面）。
func newDeckListPageWithStatus(c common, status string) deckListPage {
	return deckListPage{common: c, status: status}
}

// decksLoadedMsg 表示单词本列表加载完成。
type decksLoadedMsg struct {
	decks []model.Deck
	err   error
}

func (p deckListPage) loadDecksCmd() tea.Cmd {
	s := p.store
	return func() tea.Msg {
		decks, err := s.ListDecks()
		return decksLoadedMsg{decks: decks, err: err}
	}
}

func (p deckListPage) Init() tea.Cmd {
	return p.loadDecksCmd()
}

func (p deckListPage) Update(msg tea.Msg) (page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
	case decksLoadedMsg:
		p.err = msg.err
		p.decks = msg.decks
		if p.cursor > len(p.decks) {
			p.cursor = len(p.decks)
		}
	case tea.KeyMsg:
		return p.handleKey(msg)
	}
	return p, nil
}

func (p deckListPage) handleKey(msg tea.KeyMsg) (page, tea.Cmd) {
	maxIdx := len(p.decks) // 最后一项是 "+导入"
	switch msg.String() {
	case "q":
		return p, tea.Quit
	case "up", "k", "w":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "j", "s":
		if p.cursor < maxIdx {
			p.cursor++
		}
	case " ", "d": // 空格/d：进入背单词
		if p.cursor < len(p.decks) {
			return p, navigate(newStudyPage(p.common, p.decks[p.cursor]))
		}
	case "enter", "f":
		if p.cursor == maxIdx {
			return p, navigate(newImportPage(p.common))
		}
		return p, navigate(newSettingsPage(p.common, p.decks[p.cursor]))
	}
	return p, nil
}

func (p deckListPage) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📚 我的单词本"))
	b.WriteString("\n\n")

	if len(p.decks) == 0 {
		b.WriteString(mutedStyle.Render("还没有单词本，选择下方 + 导入一个吧"))
		b.WriteString("\n\n")
	}

	for i, d := range p.decks {
		if i == p.cursor {
			b.WriteString(selectedStyle.Render("▸ " + d.Name))
		} else {
			b.WriteString(normalStyle.Render("  " + d.Name))
		}
		b.WriteString("\n")
	}

	// "+导入" 项
	plus := "+ 导入单词本"
	if p.cursor == len(p.decks) {
		b.WriteString(selectedStyle.Render("▸ " + plus))
	} else {
		b.WriteString(normalStyle.Render("  " + plus))
	}
	b.WriteString("\n")

	if p.status != "" {
		b.WriteString("\n" + okStyle.Render(p.status))
	}

	help := "w/s 选择 · d 背单词 · f 设置/导入 · q 退出"
	b.WriteString(helpStyle.Render("\n" + help))
	return p.center(p.box().Render(b.String()))
}
