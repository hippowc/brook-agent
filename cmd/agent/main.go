package main

import (
	"context"
	"flag"
	"log"
	"time"

	"brook-agent/internal/common"
	"brook-agent/internal/core"
	"brook-agent/internal/core/memory/inmemory"
	"brook-agent/internal/core/node"
	"brook-agent/internal/core/node/llmnode"
	_ "brook-agent/internal/core/node/simplehubnode"
	_ "brook-agent/internal/core/node/toolnode"
	"brook-agent/internal/core/tool"
	_ "brook-agent/internal/core/tool/bashtool"
	_ "brook-agent/internal/core/tool/filetool"
	_ "brook-agent/internal/core/tool/networktool"
	"brook-agent/internal/entry"
	_ "brook-agent/internal/entry/cli"
	_ "brook-agent/internal/entry/http"
	"brook-agent/internal/frame"
)

func main() {
	entryName := flag.String("entry", "cli", "entry type: cli or http")
	addr := flag.String("addr", ":8080", "http listen address")
	llmBaseURL := flag.String("llm-base-url", "http://localhost:11434/v1", "OpenAI-compatible base URL")
	llmAPIKey := flag.String("llm-api-key", "ollama", "OpenAI API key")
	llmModel := flag.String("llm-model", "qwen2.5:7b", "LLM model name")
	llmTimeoutSec := flag.Int("llm-timeout-sec", 60, "LLM request timeout in seconds")
	llmTemperature := flag.Float64("llm-temperature", 0.2, "LLM temperature")
	llmSystemPrompt := flag.String("llm-system-prompt", "你是一个可靠的智能助手。", "System prompt")
	flag.Parse()

	llmnode.SetConfig(llmnode.Config{
		BaseURL:      *llmBaseURL,
		APIKey:       *llmAPIKey,
		Model:        *llmModel,
		Timeout:      time.Duration(*llmTimeoutSec) * time.Second,
		Temperature:  *llmTemperature,
		SystemPrompt: *llmSystemPrompt,
	})

	mem := inmemory.New()
	toolManager, err := tool.NewManager([]string{"bash", "file", "network"})
	if err != nil {
		log.Fatal(err)
	}

	nodeManager, err := node.NewManager([]string{"simplehub", "llm", "tool"}, node.BuildConfig{
		Memory: mem,
		Tools:  toolManager,
	})
	if err != nil {
		log.Fatal(err)
	}

	agent := &core.Engine{
		Memory:    mem,
		Nodes:     nodeManager,
		Emitter:   common.NewCompositeEmitter(common.LogEmitter{}),
		MaxRounds: 8,
	}
	handler := &frame.Engine{Agent: agent}

	ent, err := entry.New(*entryName, entry.Config{
		Name: *entryName,
		Addr: *addr,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := ent.Start(context.Background(), handler); err != nil {
		log.Fatal(err)
	}
}
