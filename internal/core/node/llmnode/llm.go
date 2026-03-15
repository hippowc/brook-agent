package llmnode

import (
	"context"
	"time"

	"brook-agent/internal/core/memory"
	"brook-agent/internal/core/node"
	"brook-agent/internal/model"
)

const Name = "llm"

// Client 抽象实际 LLM 提供方，便于后续接入 OpenAI/Ollama 等协议兼容实现。
type Client interface {
	Generate(ctx context.Context, messages []model.Message) (model.Message, error)
}

// Config 定义 llm 节点可配置项。
type Config struct {
	BaseURL      string
	APIKey       string
	Model        string
	Timeout      time.Duration
	Temperature  float64
	SystemPrompt string
}

// Node 是 llm 执行节点。
type Node struct {
	client Client
	memory memory.Store
}

var defaultConfig = Config{
	BaseURL:      "http://localhost:11434/v1",
	APIKey:       "ollama",
	Model:        "qwen2.5:7b",
	Timeout:      60 * time.Second,
	Temperature:  0.2,
	SystemPrompt: "你是一个可靠的智能助手。",
}

// SetConfig 设置 llm 节点的默认配置，需在构建节点前调用。
func SetConfig(cfg Config) {
	if cfg.BaseURL != "" {
		defaultConfig.BaseURL = cfg.BaseURL
	}
	if cfg.APIKey != "" {
		defaultConfig.APIKey = cfg.APIKey
	}
	if cfg.Model != "" {
		defaultConfig.Model = cfg.Model
	}
	if cfg.Timeout > 0 {
		defaultConfig.Timeout = cfg.Timeout
	}
	if cfg.Temperature > 0 {
		defaultConfig.Temperature = cfg.Temperature
	}
	if cfg.SystemPrompt != "" {
		defaultConfig.SystemPrompt = cfg.SystemPrompt
	}
}

func init() {
	node.Register(Name, func(cfg node.BuildConfig) node.Node {
		client := NewOpenAIClient(defaultConfig)
		return &Node{
			client: client,
			memory: cfg.Memory,
		}
	})
}

func (n *Node) Name() string { return Name }

func (n *Node) Execute(ctx context.Context, in node.Input) (node.Output, error) {
	session, err := n.memory.GetOrCreate(ctx, in.Request.SessionID)
	if err != nil {
		return node.Output{}, err
	}
	msg, err := n.client.Generate(ctx, session.Messages)
	if err != nil {
		return node.Output{}, err
	}
	if err := n.memory.SaveMessage(ctx, in.Request.SessionID, msg); err != nil {
		return node.Output{}, err
	}
	return node.Output{
		NextNode: "simplehub",
		Message:  &msg,
	}, nil
}
