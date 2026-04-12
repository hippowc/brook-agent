package agentconfig

import "testing"

func TestTryValidateModeSwitch_WithDefaults(t *testing.T) {
	r := &Root{
		Version: "1",
		Agent: AgentSpec{
			Mode:         ModeReAct,
			Name:         "a",
			Instruction:  "x",
			MaxIterations:  10,
			Tools:        ToolsSpec{},
		},
		Models: ModelsSpec{
			Providers: map[string]ProviderConfig{
				"p": {Driver: "openai"},
			},
			Active: ModelRef{Provider: "p", Model: "m"},
		},
		Memory: MemorySpec{SessionStore: "memory"},
	}

	for _, mode := range []AgentMode{
		ModeReAct, ModeDeep, ModeSequential, ModeParallel, ModeLoop,
		ModeSupervisor, ModePlanExecute,
	} {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()
			if err := TryValidateModeSwitch(r, mode); err != nil {
				t.Fatalf("TryValidateModeSwitch: %v", err)
			}
		})
	}

	t.Run("custom_rejected", func(t *testing.T) {
		if err := TryValidateModeSwitch(r, ModeCustom); err == nil {
			t.Fatal("expected error for custom mode")
		}
	})
}

func TestModeConfigYAMLMap_sequential(t *testing.T) {
	m, err := ModeConfigYAMLMap(ModeSequential)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	names, ok := m["sub_agent_names"].([]any)
	if !ok || len(names) < 2 {
		t.Fatalf("sub_agent_names: %#v", m["sub_agent_names"])
	}
}
