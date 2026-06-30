// Package ui 实现 bubbletea TUI 界面。
//
// 采用主流的「根模型 + 子页面」结构：根 App 只负责全局事件（退出、窗口尺寸）
// 与页面路由，每个页面（deckListPage / importPage / studyPage / settingsPage）
// 是独立的子模型，自行管理自己的状态、按键与渲染，并通过 navigate 命令完成跳转。
package ui

import (
	"strings"

	"simpleword/internal/store"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// page 是一个可路由的子页面。与 tea.Model 几乎一致，
// 但 Update 返回 page 而非 tea.Model，便于在页面间传递。
type page interface {
	Init() tea.Cmd
	Update(tea.Msg) (page, tea.Cmd)
	View() string
}

// common 持有所有页面共享的依赖与终端尺寸。
// 每个页面通过内嵌它来复用 store、布局辅助等。
type common struct {
	store    *store.Store
	mediaDir string
	width    int
	height   int
}

// navigateMsg 请求根 App 切换到目标页面。
type navigateMsg struct{ to page }

// navigate 返回一个切换到目标页面的命令。
func navigate(to page) tea.Cmd {
	return func() tea.Msg { return navigateMsg{to: to} }
}

// App 是根模型，负责全局事件与页面路由。
type App struct {
	common
	page page
}

// New 构造根模型，初始页面为单词本列表。
func New(s *store.Store, mediaDir string) App {
	c := common{store: s, mediaDir: mediaDir}
	return App{common: c, page: newDeckListPage(c)}
}

// Init 实现 tea.Model。
func (a App) Init() tea.Cmd {
	return a.page.Init()
}

// Update 实现 tea.Model：处理全局退出与页面路由，其余事件转发给当前页面。
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	case navigateMsg:
		a.page = msg.to
		return a, a.page.Init()
	}
	var cmd tea.Cmd
	a.page, cmd = a.page.Update(msg)
	return a, cmd
}

// View 实现 tea.Model。
func (a App) View() string {
	return a.page.View()
}

// ---- 布局辅助（页面共享）----

// center 在终端中居中显示内容。
func (c common) center(body string) string {
	if c.width == 0 || c.height == 0 {
		return body
	}
	return lipgloss.Place(c.width, c.height, lipgloss.Center, lipgloss.Center, body)
}

// box 返回一个自适应终端最大宽高的边框样式。
func (c common) box() lipgloss.Style {
	s := boxStyle
	if c.width > 0 {
		if w := c.width - 2; w > 0 {
			s = s.Width(w)
		}
	}
	if c.height > 0 {
		if h := c.height - 2; h > 0 {
			s = s.Height(h)
		}
	}
	return s
}

// withBottom 将帮助行固定在内容底部。
func (c common) withBottom(content string, bottom string) string {
	if c.height <= 0 {
		return content + "\n" + helpStyle.Render(bottom)
	}

	contentHeight := lipgloss.Height(content)
	helpHeight := lipgloss.Height(bottom)

	// boxStyle 上下边框 2 + 上下 padding 2
	innerHeight := c.height - 4
	gap := innerHeight - contentHeight - helpHeight
	if gap < 1 {
		gap = 1
	}

	return content + strings.Repeat("\n", gap) + helpStyle.Render(bottom)
}
