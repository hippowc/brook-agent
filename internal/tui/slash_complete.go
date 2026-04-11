package tui

import (
	"strings"

	"brook/pkg/agentconfig"
)

var slashTopLevel = []string{"/agent", "/config", "/new"}

func longestCommonPrefixStr(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	p := strs[0]
	for _, s := range strs[1:] {
		for len(p) > 0 && !strings.HasPrefix(s, p) {
			p = p[:len(p)-1]
		}
	}
	return p
}

// applySlashTabCompletion 对以 / 开头的输入做 Tab 补全（命令名、/agent mode、模式名）。
func (m *Model) applySlashTabCompletion() bool {
	v := m.ti.Value()
	leadingLen := len(v) - len(strings.TrimLeft(v, " \t"))
	if leadingLen < 0 {
		leadingLen = 0
	}
	leading := v[:leadingLen]
	trim := strings.TrimSpace(v)
	if trim == "" || trim == "/" {
		m.ti.SetValue(leading + "/agent mode ")
		return true
	}
	if !strings.HasPrefix(trim, "/") {
		return false
	}
	parts := strings.Fields(trim)
	if len(parts) == 0 {
		return false
	}

	// 顶层命令：/co -> /config，/a -> /agent …
	if len(parts) == 1 {
		p := parts[0]
		var hits []string
		for _, c := range slashTopLevel {
			if strings.HasPrefix(strings.ToLower(c), strings.ToLower(p)) {
				hits = append(hits, c)
			}
		}
		if len(hits) == 0 {
			return false
		}
		if len(hits) == 1 {
			if strings.EqualFold(hits[0], p) {
				if strings.EqualFold(hits[0], "/agent") {
					m.ti.SetValue(leading + "/agent mode ")
					return true
				}
				return false
			}
			m.ti.SetValue(leading + hits[0])
			return true
		}
		lcp := longestCommonPrefixStr(hits)
		if len(lcp) > len(p) {
			m.ti.SetValue(leading + lcp)
			return true
		}
		return false
	}

	if !strings.EqualFold(parts[0], "/agent") {
		return false
	}

	// /agent mode <mode>
	if len(parts) == 2 {
		if strings.EqualFold(parts[1], "mode") {
			modes := agentconfig.TabCompletableModes()
			if len(modes) == 0 {
				return false
			}
			m.ti.SetValue(leading + "/agent mode "+modes[0])
			return true
		}
		// 补全子命令 mode
		if strings.HasPrefix("mode", strings.ToLower(parts[1])) && parts[1] != "mode" {
			m.ti.SetValue(leading + "/agent mode ")
			return true
		}
		return false
	}

	if len(parts) >= 3 && strings.EqualFold(parts[1], "mode") {
		partial := strings.ToLower(parts[len(parts)-1])
		if len(parts) == 3 {
			var hits []string
			for _, mo := range agentconfig.TabCompletableModes() {
				if strings.HasPrefix(strings.ToLower(mo), partial) {
					hits = append(hits, mo)
				}
			}
			if len(hits) == 0 {
				return false
			}
			if len(hits) == 1 {
				base := strings.Join(parts[:2], " ")
				m.ti.SetValue(leading + base + " " + hits[0])
				return true
			}
			lcp := longestCommonPrefixStr(hits)
			if len(lcp) > len(partial) {
				base := strings.Join(parts[:2], " ")
				m.ti.SetValue(leading + base + " " + lcp)
				return true
			}
		}
	}
	return false
}
