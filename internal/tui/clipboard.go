package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) copyToClipboard() (tea.Model, tea.Cmd) {
	var s string
	if t := strings.TrimSpace(m.ti.Value()); t != "" {
		s = m.ti.Value()
	} else {
		s = m.transcriptPlain()
	}
	if s == "" {
		return m, nil
	}
	if err := clipboard.WriteAll(s); err != nil {
		m.turns = append(m.turns, turn{role: "meta", text: fmt.Sprintf("复制失败: %v", err)})
		m.vp.SetContent(m.transcript())
		m.vp.GotoBottom()
		return m, nil
	}
	return m, nil
}

func (m *Model) copyConfigEditorToClipboard() (tea.Model, tea.Cmd) {
	s := m.cfgTA.Value()
	if s == "" {
		return m, nil
	}
	if err := clipboard.WriteAll(s); err != nil {
		return m, nil
	}
	return m, nil
}
