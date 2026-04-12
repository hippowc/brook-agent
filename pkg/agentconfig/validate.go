package agentconfig

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Validate 对 Root 做基本一致性校验，避免运行期才暴露配置错误。
func (r *Root) Validate() error {
	if r == nil {
		return fmt.Errorf("agentconfig: root is nil")
	}
	if r.Version == "" {
		r.Version = "1"
	}
	if err := r.Agent.validate(); err != nil {
		return err
	}
	if err := r.Models.validate(); err != nil {
		return err
	}
	if r.Memory.SessionStore == "file" && strings.TrimSpace(r.Memory.SessionFilePath) == "" {
		return fmt.Errorf("agentconfig: memory.session_file_path required when session_store=file")
	}
	if r.Interrupt.Enabled && strings.EqualFold(r.Interrupt.CheckpointBackend, "file") &&
		strings.TrimSpace(r.Interrupt.CheckpointFilePath) == "" {
		return fmt.Errorf("agentconfig: interrupt.checkpoint_file_path required when checkpoint_backend=file")
	}
	if err := r.Agent.validateMode(); err != nil {
		return err
	}
	if err := r.Gateway.validate(); err != nil {
		return err
	}
	return nil
}

func (g *GatewaySpec) validate() error {
	if !g.Enabled {
		return nil
	}
	if strings.TrimSpace(g.Listen) == "" {
		g.Listen = ":8787"
	}
	mode := strings.ToLower(strings.TrimSpace(g.Auth.Mode))
	if mode == "" {
		mode = "none"
		g.Auth.Mode = "none"
	}
	switch mode {
	case "none", "bearer", "hmac":
	default:
		return fmt.Errorf("agentconfig: gateway.auth.mode must be none, bearer or hmac, got %q", g.Auth.Mode)
	}
	if mode == "bearer" && strings.TrimSpace(g.Auth.BearerTokenEnv) == "" {
		return fmt.Errorf("agentconfig: gateway.auth.bearer_token_env required when auth.mode=bearer")
	}
	if mode == "hmac" && strings.TrimSpace(g.Auth.HMACSecretEnv) == "" {
		return fmt.Errorf("agentconfig: gateway.auth.hmac_secret_env required when auth.mode=hmac")
	}
	store := strings.ToLower(strings.TrimSpace(g.Session.Store))
	if store == "" {
		store = "file"
		g.Session.Store = "file"
	}
	switch store {
	case "memory", "file":
	default:
		return fmt.Errorf("agentconfig: gateway.session.store must be memory or file, got %q", g.Session.Store)
	}
	if g.Session.FileDir != "" && !filepath.IsAbs(g.Session.FileDir) {
		return fmt.Errorf("agentconfig: gateway.session.file_dir must be absolute path, got %q", g.Session.FileDir)
	}
	if g.RateLimit != nil && g.RateLimit.Enabled {
		if g.RateLimit.RequestsPerMinute <= 0 {
			g.RateLimit.RequestsPerMinute = 120
		}
		if g.RateLimit.Burst <= 0 {
			g.RateLimit.Burst = 30
		}
	}
	return nil
}

func (a *AgentSpec) validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("agentconfig: agent.name is required")
	}
	if a.Mode == "" {
		a.Mode = ModeReAct
	}
	switch a.Mode {
	case ModeReAct, ModeDeep, ModeSequential, ModeParallel, ModeLoop, ModeSupervisor, ModePlanExecute, ModeCustom:
	default:
		return fmt.Errorf("agentconfig: unknown agent.mode %q", a.Mode)
	}
	if a.MaxIterations == 0 {
		a.MaxIterations = 20
	}
	if a.WorkingDirectory != "" && !filepath.IsAbs(a.WorkingDirectory) {
		return fmt.Errorf("agentconfig: agent.working_directory must be absolute path, got %q", a.WorkingDirectory)
	}
	if a.Tools.Filesystem != nil && a.Tools.Filesystem.Enabled {
		if a.Tools.Filesystem.Backend == "" {
			return fmt.Errorf("agentconfig: tools.filesystem.backend is required when filesystem.enabled")
		}
		if a.Tools.Filesystem.Shell && a.Tools.Filesystem.StreamingShell {
			return fmt.Errorf("agentconfig: filesystem.shell and filesystem.streaming_shell are mutually exclusive")
		}
	}
	return nil
}

func (a *AgentSpec) validateMode() error {
	switch a.Mode {
	case ModeSequential, ModeParallel, ModeLoop:
		if a.ModeConfig == nil || len(a.ModeConfig.SubAgentNames) == 0 {
			return fmt.Errorf("agentconfig: agent.mode_config.sub_agent_names required for mode %q", a.Mode)
		}
	case ModeSupervisor:
		if a.ModeConfig == nil || a.ModeConfig.Supervisor == nil ||
			strings.TrimSpace(a.ModeConfig.Supervisor.SupervisorAgentName) == "" {
			return fmt.Errorf("agentconfig: mode_config.supervisor.supervisor_agent required for supervisor mode")
		}
		if len(a.ModeConfig.SubAgentNames) == 0 {
			return fmt.Errorf("agentconfig: mode_config.sub_agent_names required for supervisor mode")
		}
	case ModePlanExecute:
		if a.ModeConfig == nil || a.ModeConfig.PlanExecute == nil {
			return fmt.Errorf("agentconfig: mode_config.plan_execute required for plan_execute mode")
		}
		pe := a.ModeConfig.PlanExecute
		if strings.TrimSpace(pe.PlannerName) == "" || strings.TrimSpace(pe.ExecutorName) == "" || strings.TrimSpace(pe.ReplannerName) == "" {
			return fmt.Errorf("agentconfig: plan_execute planner, executor, replanner names are required")
		}
	}
	return nil
}

func (m *ModelsSpec) validate() error {
	if len(m.Providers) == 0 {
		return fmt.Errorf("agentconfig: models.providers cannot be empty")
	}
	if strings.TrimSpace(m.Active.Provider) == "" || strings.TrimSpace(m.Active.Model) == "" {
		return fmt.Errorf("agentconfig: models.active.provider and models.active.model are required")
	}
	if _, ok := m.Providers[m.Active.Provider]; !ok {
		return fmt.Errorf("agentconfig: models.active.provider %q not found in providers", m.Active.Provider)
	}
	return nil
}
