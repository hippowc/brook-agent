package a2ui

import (
	"encoding/json"
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

const defaultVersion = "0.8"

// WriteAgentEvents 将 AsyncIterator 中的事件转为 JSONL 写入 w（一行一个 JSON 对象）。
func WriteAgentEvents(w io.Writer, it *adk.AsyncIterator[*adk.AgentEvent], version string) error {
	if version == "" {
		version = defaultVersion
	}
	first := true
	for {
		ev, ok := it.Next()
		if !ok {
			break
		}
		if ev == nil {
			continue
		}
		if ev.Err != nil {
			env := Envelope{Version: version, Kind: "error", Payload: map[string]any{"message": ev.Err.Error()}}
			if err := writeLine(w, &env); err != nil {
				return err
			}
			continue
		}
		if ev.Output != nil && ev.Output.MessageOutput != nil {
			mv := ev.Output.MessageOutput
			if mv.IsStreaming && mv.MessageStream != nil {
				for {
					m, err := mv.MessageStream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						env := Envelope{Version: version, Kind: "error", Payload: map[string]any{"message": err.Error()}}
						if err := writeLine(w, &env); err != nil {
							return err
						}
						break
					}
					if err := writeAssistantMessage(w, version, m, first); err != nil {
						return err
					}
					first = false
				}
				continue
			}
			if mv.Message != nil {
				if err := writeAssistantMessage(w, version, mv.Message, first); err != nil {
					return err
				}
				first = false
			}
		}
		if ev.Action != nil && ev.Action.Interrupted != nil {
			env := Envelope{
				Version: version,
				Kind:    "interrupted",
				Payload: map[string]any{"data": ev.Action.Interrupted.Data},
			}
			if err := writeLine(w, &env); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeAssistantMessage(w io.Writer, version string, m *schema.Message, surfaceNew bool) error {
	kind := "surfaceUpdate"
	if surfaceNew {
		kind = "beginRendering"
	}
	env := Envelope{
		Version: version,
		Kind:    kind,
		Payload: map[string]any{
			"role":    string(m.Role),
			"content": m.Content,
		},
	}
	return writeLine(w, &env)
}

func writeLine(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}
