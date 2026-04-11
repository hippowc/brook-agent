package agentconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandAtFileRefs 将 instruction、user_prompt 中以 @ 开头的路径替换为文件内容（通常为 Markdown）。
// 形式：整段为 "@路径" 或 "@路径" 前后可有空白；路径可为绝对路径，或与配置文件同目录的相对路径。
// configDir 为 agent.yaml 所在目录；为空时不展开（用于无路径的 LoadYAML 测试）。
func ExpandAtFileRefs(r *Root, configDir string) error {
	if r == nil || configDir == "" {
		return nil
	}
	configDir = filepath.Clean(configDir)
	var err error
	r.Agent.Instruction, err = expandAtField(r.Agent.Instruction, configDir, "agent.instruction")
	if err != nil {
		return err
	}
	r.Agent.UserPrompt, err = expandAtField(r.Agent.UserPrompt, configDir, "agent.user_prompt")
	if err != nil {
		return err
	}
	return nil
}

func expandAtField(s, configDir, field string) (string, error) {
	raw := strings.TrimSpace(s)
	if raw == "" || !strings.HasPrefix(raw, "@") {
		return s, nil
	}
	path := strings.TrimSpace(raw[1:])
	if path == "" {
		return "", fmt.Errorf("agentconfig: %s: empty path after @", field)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(configDir, path)
	}
	path = filepath.Clean(path)
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("agentconfig: %s: read %q: %w", field, path, err)
	}
	return string(b), nil
}
