// Command simpleword 是一个基于 bubbletea 的背单词 TUI 程序。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"simpleword/internal/store"
	"simpleword/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// 数据目录：~/.simpleword
	dataDir, err := appDataDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法确定数据目录:", err)
		os.Exit(1)
	}

	s, err := store.Open(filepath.Join(dataDir, "simpleword.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法打开数据库:", err)
		os.Exit(1)
	}
	defer s.Close()

	mediaDir := filepath.Join(dataDir, "media")
	m := ui.New(s, mediaDir)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "运行出错:", err)
		os.Exit(1)
	}
}

// appDataDir 返回（并确保存在）应用数据目录。
func appDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".simpleword")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
