package conversation

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestTurnsRoundTrip(t *testing.T) {
	turns := []Turn{
		{Role: "user", Text: "hello"},
		{Role: "assistant", Text: "hi", Reasoning: "think"},
		{Role: "user", Text: "tool please"},
		{Role: "assistant", Text: "", Reasoning: ""},
		{Role: "toolcall", ToolName: "fs", ToolArgs: `{"p":"/"}`, ToolID: "call-1"},
		{Role: "toolresult", ToolName: "fs", Text: "ok"},
		{Role: "assistant", Text: "done"},
	}
	msgs := TurnsToMessages(turns, 0)
	if len(msgs) < 4 {
		t.Fatalf("messages len got %d", len(msgs))
	}
	back := MessagesToTurns(msgs)
	if len(back) < 4 {
		t.Fatalf("turns len got %d", len(back))
	}
}

func TestTrimOrphanTool(t *testing.T) {
	msgs := []*schema.Message{
		schema.UserMessage("a"),
		schema.AssistantMessage("b", nil),
		schema.ToolMessage("x", "id1", schema.WithToolName("t")),
	}
	out := trimMessageTail(msgs, 1)
	if len(out) != 0 && out[0].Role == schema.Tool {
		t.Fatalf("should not start with tool")
	}
}
