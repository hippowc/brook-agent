// Package model 根据 agentconfig 构造 eino ToolCallingChatModel（经各厂商 ChatModel 实现）。
package model

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/cloudwego/eino/components/model"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	openaiext "github.com/cloudwego/eino-ext/components/model/openai"

	"github.com/hippowc/brook/pkg/agentconfig"
)

// NewChatModel 按 provider 的 driver 创建模型实例；当前支持 openai、ollama。
func NewChatModel(ctx context.Context, root *agentconfig.Root) (model.ToolCallingChatModel, error) {
	p, ok := root.Models.Providers[root.Models.Active.Provider]
	if !ok {
		return nil, fmt.Errorf("model: unknown provider %q", root.Models.Active.Provider)
	}
	switch p.Driver {
	case "openai":
		return newOpenAI(ctx, p, root.Models.Active.Model)
	case "ollama":
		return newOllama(ctx, p, root.Models.Active.Model)
	default:
		return nil, fmt.Errorf("model: unsupported driver %q", p.Driver)
	}
}

func newOpenAI(ctx context.Context, p agentconfig.ProviderConfig, modelName string) (model.ToolCallingChatModel, error) {
	key := ""
	if p.APIKeyEnv != "" {
		key = os.Getenv(p.APIKeyEnv)
	}
	if key == "" && p.APIKeyEnv != "" {
		return nil, fmt.Errorf("model: environment %s is empty", p.APIKeyEnv)
	}
	cfg := &openaiext.ChatModelConfig{
		APIKey: key,
		Model:  modelName,
		BaseURL: p.BaseURL,
	}
	if p.Extra != nil {
		if v, ok := p.Extra["temperature"]; ok {
			if f, ok := toFloat32(v); ok {
				cfg.Temperature = &f
			}
		}
		if v, ok := p.Extra["by_azure"]; ok {
			if b, ok := v.(bool); ok {
				cfg.ByAzure = b
			}
		}
	}
	cm, err := openaiext.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func newOllama(ctx context.Context, p agentconfig.ProviderConfig, modelName string) (model.ToolCallingChatModel, error) {
	base := p.BaseURL
	if base == "" {
		base = "http://127.0.0.1:11434"
	}
	cfg := &ollama.ChatModelConfig{
		BaseURL: base,
		Model:   modelName,
	}
	cm, err := ollama.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func toFloat32(v any) (float32, bool) {
	switch t := v.(type) {
	case float64:
		return float32(t), true
	case float32:
		return t, true
	case int:
		return float32(t), true
	case string:
		f, err := strconv.ParseFloat(t, 32)
		if err != nil {
			return 0, false
		}
		return float32(f), true
	default:
		return 0, false
	}
}
