package ui

import (
	"fmt"
	"strings"

	"simpleword/internal/anki"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// importPage 负责输入 .apkg 路径并导入单词本。
type importPage struct {
	common
	input  textinput.Model
	status string
}

func newImportPage(c common) importPage {
	ti := textinput.New()
	ti.Placeholder = "输入 .apkg 文件路径"
	ti.CharLimit = 512
	ti.Width = 50
	ti.Focus()
	return importPage{common: c, input: ti}
}

// importDoneMsg 表示一次导入操作完成。
type importDoneMsg struct {
	name  string
	count int
	err   error
}

func (p importPage) importCmd(path string) tea.Cmd {
	s, mediaDir := p.store, p.mediaDir
	return func() tea.Msg {
		res, err := anki.Parse(path, mediaDir)
		if err != nil {
			return importDoneMsg{err: err}
		}
		deckID, err := s.CreateDeck(res.Name)
		if err != nil {
			return importDoneMsg{err: err}
		}
		if err := s.AddWords(deckID, res.Words); err != nil {
			return importDoneMsg{err: err}
		}
		return importDoneMsg{name: res.Name, count: len(res.Words)}
	}
}

func (p importPage) Init() tea.Cmd {
	return textinput.Blink
}

func (p importPage) Update(msg tea.Msg) (page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		return p, nil

	case importDoneMsg:
		if msg.err != nil {
			return p, navigate(newDeckListPageWithStatus(p.common, "导入失败: "+msg.err.Error()))
		}
		status := fmt.Sprintf("已导入「%s」，共 %d 个单词", msg.name, msg.count)
		return p, navigate(newDeckListPageWithStatus(p.common, status))

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return p, navigate(newDeckListPage(p.common))
		case "enter":
			path := strings.TrimSpace(p.input.Value())
			if path == "" {
				return p, nil
			}
			p.status = "正在导入..."
			return p, p.importCmd(path)
		}
	}

	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

func (p importPage) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📥 导入 Anki 单词本"))
	b.WriteString("\n\n")
	b.WriteString(p.input.View())
	if p.status != "" {
		b.WriteString("\n\n" + mutedStyle.Render(p.status))
	}
	body := p.withBottom(b.String(), "enter 确认导入 · esc 取消")
	return p.center(p.box().Render(body))
}
