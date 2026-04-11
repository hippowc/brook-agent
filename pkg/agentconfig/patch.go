package agentconfig

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PatchAgentModeInYAMLFile 仅修改磁盘 YAML 中 agent.mode，其它键保持 unmarshaled 后的内容（顺序可能变化）。
func PatchAgentModeInYAMLFile(path string, mode AgentMode) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("agentconfig: parse yaml: %w", err)
	}
	agent, ok := doc["agent"].(map[string]any)
	if !ok {
		return fmt.Errorf("agentconfig: missing or invalid agent section")
	}
	agent["mode"] = string(mode)
	doc["agent"] = agent
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// TryValidateModeSwitch 在仅修改 mode 的情况下校验配置是否仍合法（用于切换前检查）。
func TryValidateModeSwitch(r *Root, mode AgentMode) error {
	if r == nil {
		return fmt.Errorf("agentconfig: nil root")
	}
	cp := *r
	cp.Agent.Mode = mode
	return cp.Validate()
}
