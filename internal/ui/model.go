// Package ui 实现 bubbletea TUI 界面（MVP 中的 View + Presenter 层）。
package ui

import (
	"fmt"
	"strings"

	"simpleword/internal/anki"
	"simpleword/internal/audio"
	"simpleword/internal/model"
	"simpleword/internal/store"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// screen 表示当前界面状态。
type screen int

const (
	screenDeckList      screen = iota // 主界面：单词本列表
	screenImport                      // 导入：输入 apkg 路径
	screenStudy                       // 背单词
	screenSettings                    // 单词本设置
	screenCongrats                    // 祝贺框
	screenDeleteConfirm               // 删除确认
)

// roundSize 是每一轮抽取的单词数。
const roundSize = 7

// Model 是顶层 bubbletea 模型。
type Model struct {
	store    *store.Store
	mediaDir string

	screen screen
	width  int
	height int
	err    error
	status string

	// 主界面
	decks  []model.Deck
	cursor int // 0..len(decks) 为单词本，len(decks) 为 "+导入"

	// 导入
	input textinput.Model

	// 背单词
	studyDeck model.Deck
	round     []model.Word
	roundIdx  int
	showDef   bool
	defScroll int

	// 设置
	settingsDeck   model.Deck
	settingsStats  model.DeckStats
	settingsCursor int // 0: 开始学习, 1: 删除单词本
}

// New 构造初始模型。
func New(s *store.Store, mediaDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "输入 .apkg 文件路径"
	ti.CharLimit = 512
	ti.Width = 50

	m := Model{
		store:    s,
		mediaDir: mediaDir,
		screen:   screenDeckList,
		input:    ti,
	}
	return m
}

// Init 实现 tea.Model。
func (m Model) Init() tea.Cmd {
	return loadDecksCmd(m.store)
}

// ---- 消息类型 ----

type decksLoadedMsg struct {
	decks []model.Deck
	err   error
}

type importDoneMsg struct {
	name  string
	count int
	err   error
}

type roundLoadedMsg struct {
	deck  model.Deck
	words []model.Word
	err   error
}

type statsLoadedMsg struct {
	deck  model.Deck
	stats model.DeckStats
	err   error
}

type deckDeletedMsg struct{ err error }

// ---- 命令 ----

func loadDecksCmd(s *store.Store) tea.Cmd {
	return func() tea.Msg {
		decks, err := s.ListDecks()
		return decksLoadedMsg{decks: decks, err: err}
	}
}

func importCmd(s *store.Store, path, mediaDir string) tea.Cmd {
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

func loadRoundCmd(s *store.Store, deck model.Deck) tea.Cmd {
	return func() tea.Msg {
		words, err := s.PickRound(deck.ID, roundSize)
		return roundLoadedMsg{deck: deck, words: words, err: err}
	}
}

func loadStatsCmd(s *store.Store, deck model.Deck) tea.Cmd {
	return func() tea.Msg {
		st, err := s.Stats(deck.ID)
		return statsLoadedMsg{deck: deck, stats: st, err: err}
	}
}

func deleteDeckCmd(s *store.Store, id int64) tea.Cmd {
	return func() tea.Msg {
		return deckDeletedMsg{err: s.DeleteDeck(id)}
	}
}

// Update 实现 tea.Model。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case decksLoadedMsg:
		m.err = msg.err
		m.decks = msg.decks
		if m.cursor > len(m.decks) {
			m.cursor = len(m.decks)
		}
		return m, nil

	case importDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = "导入失败: " + msg.err.Error()
		} else {
			m.status = fmt.Sprintf("已导入「%s」，共 %d 个单词", msg.name, msg.count)
		}
		m.screen = screenDeckList
		m.input.Blur()
		m.input.SetValue("")
		return m, loadDecksCmd(m.store)

	case roundLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.screen = screenDeckList
			return m, nil
		}
		m.studyDeck = msg.deck
		m.round = msg.words
		m.roundIdx = 0
		m.showDef = false
		m.defScroll = 0
		if len(m.round) == 0 {
			m.status = "该单词本暂无单词"
			m.screen = screenDeckList
			return m, nil
		}
		m.screen = screenStudy
		return m, playCurrentCmd(m)

	case statsLoadedMsg:
		m.err = msg.err
		m.settingsDeck = msg.deck
		m.settingsStats = msg.stats
		m.settingsCursor = 0
		m.screen = screenSettings
		return m, nil

	case deckDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.status = "已删除单词本"
		}
		m.screen = screenDeckList
		m.cursor = 0
		return m, loadDecksCmd(m.store)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// playCurrentCmd 播放当前单词语音（副作用命令）。
func playCurrentCmd(m Model) tea.Cmd {
	return func() tea.Msg {
		if m.roundIdx < len(m.round) {
			audio.Play(m.round[m.roundIdx].Audio)
		}
		return nil
	}
}

// handleKey 根据当前界面分发按键。
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 导入界面优先把按键交给输入框
	if m.screen == screenImport {
		switch msg.String() {
		case "esc":
			m.screen = screenDeckList
			m.input.Blur()
			m.input.SetValue("")
			return m, nil
		case "enter":
			path := strings.TrimSpace(m.input.Value())
			if path == "" {
				return m, nil
			}
			m.status = "正在导入..."
			return m, importCmd(m.store, path, m.mediaDir)
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.screen == screenDeckList {
			return m, tea.Quit
		}
	}

	switch m.screen {
	case screenDeckList:
		return m.keyDeckList(msg)
	case screenStudy:
		return m.keyStudy(msg)
	case screenSettings:
		return m.keySettings(msg)
	case screenCongrats:
		return m.keyCongrats(msg)
	case screenDeleteConfirm:
		return m.keyDeleteConfirm(msg)
	}
	return m, nil
}

func (m Model) keyDeckList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxIdx := len(m.decks) // 最后一项是 "+导入"
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < maxIdx {
			m.cursor++
		}
	case " ": // 空格：进入背单词
		if m.cursor < len(m.decks) {
			m.status = ""
			return m, loadRoundCmd(m.store, m.decks[m.cursor])
		}
	case "enter":
		if m.cursor == maxIdx {
			// +导入
			m.screen = screenImport
			m.input.SetValue("")
			m.input.Focus()
			return m, textinput.Blink
		}
		// 单词本设置
		return m, loadStatsCmd(m.store, m.decks[m.cursor])
	}
	return m, nil
}

func (m Model) keyStudy(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.screen = screenDeckList
		return m, loadDecksCmd(m.store)
	case "s": // 查看完整释义
		m.showDef = true
		m.defScroll = 0
		return m, nil
	case "up", "k": // 释义向上滚动
		if m.showDef && m.defScroll > 0 {
			m.defScroll--
		}
		return m, nil
	case "down", "j": // 释义向下滚动
		if m.showDef && m.defScroll < m.maxDefScroll() {
			m.defScroll++
		}
		return m, nil
	case "a": // 上一个
		if m.roundIdx > 0 {
			m.roundIdx--
			m.showDef = false
			m.defScroll = 0
			return m, playCurrentCmd(m)
		}
		return m, nil
	case "d": // 掌握，进入下一个
		cur := m.round[m.roundIdx]
		cmd := func() tea.Msg { m.store.MarkMastered(cur.ID); return nil }
		m.round[m.roundIdx].Mastered = true
		return m.advance(cmd)
	case " ", "enter": // 仍需复习，进入下一个（不标记掌握）
		return m.advance(nil)
	}
	return m, nil
}

// advance 前进到下一个未掌握的单词；若本轮全部掌握则弹出祝贺框。
func (m Model) advance(extra tea.Cmd) (tea.Model, tea.Cmd) {
	m.showDef = false
	m.defScroll = 0
	// 从当前位置向后寻找下一个未掌握单词，循环回到开头。
	n := len(m.round)
	for i := 1; i <= n; i++ {
		idx := (m.roundIdx + i) % n
		if !m.round[idx].Mastered {
			m.roundIdx = idx
			return m, tea.Batch(extra, playCurrentCmd(m))
		}
	}
	// 全部掌握
	m.screen = screenCongrats
	return m, extra
}

func (m Model) keyCongrats(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "y", " ": // 继续学习：拉取下一轮
		return m, loadRoundCmd(m.store, m.studyDeck)
	case "esc", "n", "q": // 返回主界面
		m.screen = screenDeckList
		return m, loadDecksCmd(m.store)
	}
	return m, nil
}

func (m Model) keySettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.screen = screenDeckList
		return m, nil
	case "up", "k":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
	case "down", "j":
		if m.settingsCursor < 1 {
			m.settingsCursor++
		}
	case "enter", " ":
		if m.settingsCursor == 0 {
			return m, loadRoundCmd(m.store, m.settingsDeck)
		}
		m.screen = screenDeleteConfirm
	}
	return m, nil
}

func (m Model) keyDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, deleteDeckCmd(m.store, m.settingsDeck.ID)
	case "n", "esc", "q":
		m.screen = screenSettings
	}
	return m, nil
}
