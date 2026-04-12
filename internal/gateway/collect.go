package gateway

import (
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// CollectAssistantText 从 Agent 事件流中拼接 assistant 角色的文本（含流式分片）。
func CollectAssistantText(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var b []byte
	for {
		ev, ok := iter.Next()
		if !ok {
			break
		}
		if ev == nil {
			continue
		}
		if ev.Err != nil {
			return string(b), ev.Err
		}
		if ev.Output == nil || ev.Output.MessageOutput == nil {
			continue
		}
		mv := ev.Output.MessageOutput
		if mv.Role != schema.Assistant {
			continue
		}
		if mv.IsStreaming && mv.MessageStream != nil {
			for {
				msg, err := mv.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					return string(b), err
				}
				if msg != nil && msg.Content != "" {
					b = append(b, msg.Content...)
				}
			}
			continue
		}
		msg, err := mv.GetMessage()
		if err != nil {
			return string(b), err
		}
		if msg != nil && msg.Content != "" {
			b = append(b, msg.Content...)
		}
	}
	return string(b), nil
}
