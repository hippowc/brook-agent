package agentconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandAtFileRefs(t *testing.T) {
	dir := t.TempDir()
	prompt := filepath.Join(dir, "p.md")
	if err := os.WriteFile(prompt, []byte("# hi\nbody"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &Root{
		Version: "1",
		Agent: AgentSpec{
			Name:          "t",
			Instruction:   "@" + filepath.Base(prompt),
			UserPrompt:    "@" + filepath.Base(prompt),
			Mode:          ModeReAct,
		},
		Memory: MemorySpec{SessionStore: "memory"},
		Models: ModelsSpec{
			Providers: map[string]ProviderConfig{
				"o": {Driver: "openai", APIKeyEnv: "K", Extra: map[string]any{}},
			},
			Active: ModelRef{Provider: "o", Model: "m"},
		},
		Interrupt: InterruptSpec{Enabled: false},
	}
	if err := ExpandAtFileRefs(r, dir); err != nil {
		t.Fatal(err)
	}
	if r.Agent.Instruction != "# hi\nbody" {
		t.Fatalf("instruction: %q", r.Agent.Instruction)
	}
}
