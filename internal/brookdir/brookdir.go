// Package brookdir 定义用户主目录下 ~/.brook 的数据布局（配置、会话、checkpoint、日志等）。
package brookdir

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed default_agent.yaml
var defaultAgentYAML string

const (
	DirName             = ".brook"
	AgentFileName       = "agent.yaml"
	WorkspaceName       = "workspace"
	CheckpointName      = "checkpoints"
	ConversationsName   = "conversations"
	GatewaySessionsName = "gateway/sessions"
	SessionName         = "session.json"
	LogFileName         = "brook.log"
	CurrentConversation = "current_conversation"
)

// Root 返回 ~/.brook 的绝对路径。
func Root() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("brookdir: home: %w", err)
	}
	return filepath.Join(home, DirName), nil
}

// AgentYAML 返回默认配置文件路径 ~/.brook/agent.yaml。
func AgentYAML() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, AgentFileName), nil
}

// ConversationsDir 返回 ~/.brook/conversations（多轮对话存档目录）。
func ConversationsDir() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, ConversationsName), nil
}

// ConversationFile 返回 ~/.brook/conversations/<id>.json（id 应为合法 UUID，调用方负责校验）。
func ConversationFile(id string) (string, error) {
	dir, err := ConversationsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

// CurrentConversationFile 返回记录「当前默认会话 ID」的文件路径 ~/.brook/current_conversation。
func CurrentConversationFile() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, CurrentConversation), nil
}

// ReadCurrentConversationID 读取 ~/.brook/current_conversation 中的 UUID；文件不存在时返回空字符串。
func ReadCurrentConversationID() (string, error) {
	p, err := CurrentConversationFile()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// WriteCurrentConversationID 写入当前默认会话 ID（单行 UUID）。
func WriteCurrentConversationID(id string) error {
	p, err := CurrentConversationFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(strings.TrimSpace(id)+"\n"), 0o600)
}

// GatewaySessionsDir 返回 ~/.brook/gateway/sessions（brook-gateway 按用户隔离的 session KV 文件目录）。
func GatewaySessionsDir() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, GatewaySessionsName), nil
}

// LogFile 返回日志文件路径 ~/.brook/brook.log。
func LogFile() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, LogFileName), nil
}

// Ensure 创建 ~/.brook 目录结构；若不存在 agent.yaml 则写入内置默认配置（路径占位符会替换为绝对路径）。
// 返回应加载的 agent.yaml 绝对路径。
func Ensure() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	workspace := filepath.Join(root, WorkspaceName)
	chk := filepath.Join(root, CheckpointName)
	conv := filepath.Join(root, ConversationsName)
	gwSessDir := filepath.Join(root, GatewaySessionsName)
	for _, d := range []string{root, workspace, chk, conv, gwSessDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", fmt.Errorf("brookdir: mkdir %s: %w", d, err)
		}
	}

	cfgPath := filepath.Join(root, AgentFileName)
	if _, err := os.Stat(cfgPath); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		sessionPath := filepath.Join(root, SessionName)
		body := strings.NewReplacer(
			"__WORKSPACE__", workspace,
			"__SESSION__", sessionPath,
			"__CHECKPOINTS__", chk,
		).Replace(defaultAgentYAML)
		if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
			return "", fmt.Errorf("brookdir: write default config: %w", err)
		}
	}

	abs, err := filepath.Abs(cfgPath)
	if err != nil {
		return cfgPath, nil
	}
	return abs, nil
}
