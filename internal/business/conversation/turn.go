// Package conversation 持久化 TUI 与 ADK 之间的多轮对话上下文（schema.Message 序列化）。
package conversation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// Turn 与 TUI transcript 一一对应，用于在 UI 与 schema.Message 之间转换。
type Turn struct {
	Role      string
	Text      string
	Reasoning string
	Stream    bool

	ToolName string
	ToolArgs string
	ToolID   string
}

var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidateID 防止路径穿越与非 UUID 文件名。
func ValidateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("conversation: empty id")
	}
	if !uuidRe.MatchString(id) {
		return fmt.Errorf("conversation: id must be a UUID")
	}
	return nil
}

// TurnsToMessages 将 TUI 轮次转为 ADK Run 使用的消息序列（含 user / assistant / tool）。
// 跳过 error、meta；跳过仍处流式未结束的 assistant。maxMessages>0 时对尾部做条数裁剪并尽量不以孤立 tool 消息开头。
func TurnsToMessages(turns []Turn, maxMessages int) []*schema.Message {
	var msgs []*schema.Message
	i := 0
	for i < len(turns) {
		t := turns[i]
		switch t.Role {
		case "error", "meta":
			i++
			continue
		case "user":
			msgs = append(msgs, schema.UserMessage(t.Text))
			i++
		case "assistant":
			if t.Stream {
				i++
				continue
			}
			content := t.Text
			reasoning := t.Reasoning
			i++
			var calls []schema.ToolCall
			for i < len(turns) && turns[i].Role == "toolcall" {
				tc := turns[i]
				calls = append(calls, schema.ToolCall{
					ID:   tc.ToolID,
					Type: "function",
					Function: schema.FunctionCall{
						Name:      tc.ToolName,
						Arguments: tc.ToolArgs,
					},
				})
				i++
			}
			am := schema.AssistantMessage(content, calls)
			am.ReasoningContent = reasoning
			msgs = append(msgs, am)
			for _, tc := range calls {
				if i >= len(turns) || turns[i].Role != "toolresult" {
					break
				}
				tr := turns[i]
				toolID := tc.ID
				msgs = append(msgs, schema.ToolMessage(tr.Text, toolID, schema.WithToolName(tr.ToolName)))
				i++
			}
		case "toolcall", "toolresult":
			// 孤立片段（例如旧数据损坏），跳过以免打乱模型上下文
			i++
		default:
			i++
		}
	}
	msgs = trimMessageTail(msgs, maxMessages)
	return msgs
}

// MessagesToTurns 将存档中的消息还原为 TUI 可渲染的轮次。
func MessagesToTurns(msgs []*schema.Message) []Turn {
	var turns []Turn
	for _, m := range msgs {
		if m == nil {
			continue
		}
		switch m.Role {
		case schema.User:
			turns = append(turns, Turn{Role: "user", Text: m.Content})
		case schema.Assistant:
			turns = append(turns, Turn{
				Role:      "assistant",
				Text:      m.Content,
				Reasoning: m.ReasoningContent,
				Stream:    false,
			})
			for _, tc := range m.ToolCalls {
				turns = append(turns, Turn{
					Role:     "toolcall",
					ToolName: tc.Function.Name,
					ToolArgs: tc.Function.Arguments,
					ToolID:   tc.ID,
				})
			}
		case schema.Tool:
			turns = append(turns, Turn{
				Role:     "toolresult",
				ToolName: m.ToolName,
				Text:     m.Content,
			})
		default:
			// system 等暂不展示于当前 TUI
		}
	}
	return turns
}

func trimMessageTail(msgs []*schema.Message, max int) []*schema.Message {
	if max <= 0 || len(msgs) <= max {
		return msgs
	}
	out := msgs[len(msgs)-max:]
	// 避免以 tool 开头导致模型上下文不完整
	for len(out) > 0 && out[0].Role == schema.Tool {
		out = out[1:]
	}
	return out
}
