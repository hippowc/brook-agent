package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// 终端默认背景，仅前景色与少量分隔线（避免整块铺色显得杂乱）。
var (
	colorMuted     = lipgloss.Color("245")
	colorDim       = lipgloss.Color("238")
	colorText      = lipgloss.Color("252")
	colorEmphasis  = lipgloss.Color("255")
	colorAccent    = lipgloss.Color("214")
	colorReasoning = lipgloss.Color("240")
	colorTool      = lipgloss.Color("109")
	colorBorder    = lipgloss.Color("236")
	colorErr       = lipgloss.Color("203")
)

var (
	// 外层仅轻微水平留白，不铺背景
	styleApp = lipgloss.NewStyle().Padding(0, 1)

	styleHeader = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 0, 1, 0)

	styleTitle = lipgloss.NewStyle().
			Foreground(colorEmphasis).
			Bold(true)

	styleBadge = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleFooter = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(1, 0, 0, 0)

	styleUserLabel = lipgloss.NewStyle().Foreground(colorMuted)
	styleUserText  = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 0, 0, 2)

	styleAsstLabel = lipgloss.NewStyle().Foreground(colorAccent)
	// 主回复：略亮
	styleAsstBody = lipgloss.NewStyle().
			Foreground(colorEmphasis).
			Padding(0, 0, 0, 2).
			Margin(0, 0, 1, 0)
	// 模型思考 / reasoning：明显更淡
	styleReasoning = lipgloss.NewStyle().
			Foreground(colorReasoning).
			Italic(true).
			Padding(0, 0, 0, 2).
			Margin(0, 0, 1, 0)

	styleMeta = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	styleErr = lipgloss.NewStyle().
			Foreground(colorErr).
			Margin(0, 0, 1, 0)

	styleToolHeader = lipgloss.NewStyle().
			Foreground(colorTool).
			Bold(true)

	styleToolArgs = lipgloss.NewStyle().
			Foreground(colorMuted).
			BorderLeft(true).
			BorderForeground(colorDim).
			Padding(0, 0, 0, 2).
			Margin(0, 0, 1, 0)

	styleToolResultLabel = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleToolResultBody = lipgloss.NewStyle().
			Foreground(colorText).
			BorderLeft(true).
			BorderForeground(colorDim).
			Padding(0, 0, 0, 2).
			Margin(0, 0, 1, 1)

	styleMetaNote = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Margin(0, 0, 1, 0)

	// 输入区上下两根横线（Claude Code 式）
	styleInputRule = lipgloss.NewStyle().Foreground(colorBorder)

	styleTranscriptHint = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
)
