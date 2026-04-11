package tui

import "strings"

// transcriptPlain 生成无 ANSI 的纯文本，供复制到剪贴板。
func (m *Model) transcriptPlain() string {
	var b strings.Builder
	for i, t := range m.turns {
		isLast := i == len(m.turns)-1
		switch t.role {
		case "user":
			b.WriteString("You: ")
			b.WriteString(t.text)
			b.WriteByte('\n')
		case "assistant":
			rs := t.reasoning
			tx := t.text
			if isLast && t.stream {
				rs += m.pendingReasoning.String()
				tx += m.pending.String()
			}
			if strings.TrimSpace(rs) != "" {
				b.WriteString("thinking:\n")
				b.WriteString(strings.TrimRight(rs, "\n"))
				b.WriteString("\n\n")
			}
			b.WriteString("Brook: ")
			b.WriteString(tx)
			b.WriteByte('\n')
		case "toolcall":
			b.WriteString("tool call ")
			b.WriteString(t.toolName)
			if t.toolID != "" {
				b.WriteString(" (")
				b.WriteString(t.toolID)
				b.WriteString(")")
			}
			b.WriteString("\n")
			b.WriteString(t.toolArgs)
			b.WriteByte('\n')
		case "toolresult":
			b.WriteString("tool result ")
			b.WriteString(t.toolName)
			b.WriteString(":\n")
			b.WriteString(t.text)
			b.WriteByte('\n')
		case "meta", "error":
			b.WriteString(t.text)
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}
