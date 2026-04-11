package agentconfig

import "testing"

func TestLoadYAML_Minimal(t *testing.T) {
	y := `
version: "1"
agent:
  mode: react
  name: "t"
  instruction: "sys"
models:
  providers:
    p1:
      driver: openai
      api_key_env: K
  active:
    provider: p1
    model: m1
`
	r, err := LoadYAML([]byte(y))
	if err != nil {
		t.Fatal(err)
	}
	if r.Agent.MaxIterations != 20 {
		t.Fatalf("default max_iterations: got %d", r.Agent.MaxIterations)
	}
}

func TestValidate_WorkdirRelative(t *testing.T) {
	y := `
version: "1"
agent:
  mode: react
  name: "t"
  instruction: "sys"
  working_directory: "relative/path"
models:
  providers:
    p1:
      driver: openai
  active:
    provider: p1
    model: m1
`
	_, err := LoadYAML([]byte(y))
	if err == nil {
		t.Fatal("expected error for non-absolute working_directory")
	}
}
