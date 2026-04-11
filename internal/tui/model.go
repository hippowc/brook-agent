package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"brook/internal/brookdir"
	"brook/internal/business/conversation"
	"brook/internal/launcher"
	"brook/pkg/agentconfig"
)

// streamMsg 表示模型输出增量、工具调用、工具结果或结束。
type streamMsg struct {
	runID     int
	text      string
	reasoning string
	err       error
	done      bool
	cancelled bool
	// session 为单次 run 结束时 ADK runSession 中的 KV 快照（如 output_key），供写回文件。
	session map[string]any

	toolCalls []schema.ToolCall

	toolResultName string
	toolResultBody string
}

type turn struct {
	role   string // user | assistant | error | toolcall | toolresult | meta
	text   string
	// reasoning 模型思考过程（如 ReasoningContent），与主回复区分展示
	reasoning string
	stream    bool

	toolName string
	toolArgs string
	toolID   string
}

// Model Bubble Tea 根模型。
type Model struct {
	rt     *launcher.Runtime
	width  int
	height int

	vp viewport.Model
	ti textinput.Model

	turns   []turn
	pending          strings.Builder
	pendingReasoning strings.Builder

	busy bool
	ch   chan streamMsg

	runID     int
	runCancel context.CancelFunc

	agentName string
	metaLine  string

	cfgPath       string
	editingConfig bool
	cfgTA         textarea.Model

	// 多轮对话持久化（~/.brook/conversations/<uuid>.json）
	convID       string
	convFilePath string
}

const maxToolArgRunes = 8000
const maxToolBodyRunes = 500000

// New 构建 TUI；cfgPath 为 YAML 路径；convID 为合法 UUID，对应该会话的存档文件。
func New(rt *launcher.Runtime, cfgPath, convID string) *Model {
	ti := textinput.New()
	ti.Placeholder = "输入消息…  /agent mode  ·  /config  ·  /new  ·  Tab 补全"
	ti.CharLimit = 8192
	ti.Width = 80
	ti.Prompt = ""
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)

	cfgTA := textarea.New()
	cfgTA.ShowLineNumbers = false
	cfgTA.CharLimit = 512 * 1024
	cfgTA.Prompt = ""
	cfgTA.FocusedStyle.Base = lipgloss.NewStyle().Foreground(colorText)
	cfgTA.BlurredStyle.Base = lipgloss.NewStyle().Foreground(colorMuted)
	cfgTA.FocusedStyle.Text = lipgloss.NewStyle().Foreground(colorEmphasis)
	cfgTA.BlurredStyle.Text = lipgloss.NewStyle().Foreground(colorMuted)

	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true

	mo := &Model{
		rt:        rt,
		ti:        ti,
		vp:        vp,
		agentName: rt.Root.Agent.Name,
		ch:        make(chan streamMsg, 128),
		cfgPath:   cfgPath,
		cfgTA:     cfgTA,
		convID:    convID,
	}
	cfp, err := brookdir.ConversationFile(convID)
	if err == nil {
		mo.convFilePath = cfp
	}
	mo.refreshMetaLine()
	mo.loadConversationFile()
	// 必须在返回的模型上同步 Focus：Init 里若用值接收器调用 Focus 只会改副本，会导致输入区不聚焦。
	_ = mo.ti.Focus()
	return mo
}

func (m *Model) refreshMetaLine() {
	short := m.convID
	if len(short) > 8 {
		short = short[:8]
	}
	m.metaLine = fmt.Sprintf("%s · %s/%s · %s", m.cfgPath,
		m.rt.Root.Models.Active.Provider, m.rt.Root.Models.Active.Model, short)
}

func (m *Model) loadConversationFile() {
	if m.convFilePath == "" {
		return
	}
	f, err := conversation.Load(m.convFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("加载会话: %v", err)})
		return
	}
	if f.ID != "" && f.ID != m.convID {
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("会话文件 id 与参数不一致: file=%s arg=%s", f.ID, m.convID)})
		return
	}
	for _, ct := range conversation.MessagesToTurns(f.MessagesPointers()) {
		m.turns = append(m.turns, convTurnToUI(ct))
	}
}

func convTurnToUI(ct conversation.Turn) turn {
	return turn{
		role:      ct.Role,
		text:      ct.Text,
		reasoning: ct.Reasoning,
		stream:    false,
		toolName:  ct.ToolName,
		toolArgs:  ct.ToolArgs,
		toolID:    ct.ToolID,
	}
}

func (m *Model) turnToConv(t turn) conversation.Turn {
	return conversation.Turn{
		Role:      t.role,
		Text:      t.text,
		Reasoning: t.reasoning,
		Stream:    t.stream,
		ToolName:  t.toolName,
		ToolArgs:  t.toolArgs,
		ToolID:    t.toolID,
	}
}

func (m *Model) persistableConvTurns() []conversation.Turn {
	var out []conversation.Turn
	for _, t := range m.turns {
		if t.role == "error" || t.role == "meta" {
			continue
		}
		if t.role == "assistant" && t.stream {
			continue
		}
		out = append(out, m.turnToConv(t))
	}
	return out
}

func (m *Model) historyTurnsForRun() []conversation.Turn {
	if len(m.turns) < 2 {
		return nil
	}
	prev := m.turns[:len(m.turns)-2]
	var out []conversation.Turn
	for _, t := range prev {
		if t.role == "error" || t.role == "meta" {
			continue
		}
		if t.role == "assistant" && t.stream {
			continue
		}
		out = append(out, m.turnToConv(t))
	}
	return out
}

func conversationPreview(turns []turn) string {
	for _, t := range turns {
		if t.role == "user" && strings.TrimSpace(t.text) != "" {
			s := strings.TrimSpace(t.text)
			if len(s) > 80 {
				return s[:80] + "…"
			}
			return s
		}
	}
	return ""
}

// saveConversation 将当前可持久化轮次写入 ~/.brook/conversations/<uuid>.json。
func (m *Model) saveConversation() error {
	if m.convFilePath == "" || m.convID == "" {
		return nil
	}
	msgs := conversation.TurnsToMessages(m.persistableConvTurns(), 0)
	f := &conversation.File{ID: m.convID, ConfigPath: m.cfgPath}
	f.SetFromMessages(msgs)
	if err := conversation.Save(m.convFilePath, f); err != nil {
		return err
	}
	_ = brookdir.WriteCurrentConversationID(m.convID)
	if convDir, err := brookdir.ConversationsDir(); err == nil {
		_ = conversation.UpdateIndex(convDir, m.convID, m.cfgPath, conversationPreview(m.turns))
	}
	return nil
}

// SaveConversation 供进程退出时调用，与 SessionValues 落盘并列。
func (m *Model) SaveConversation() error {
	return m.saveConversation()
}

// ConversationID 返回当前会话 UUID。
func (m *Model) ConversationID() string {
	return m.convID
}

// Runtime 返回当前加载的 launcher 运行时（配置热重载后会替换），供退出时保存 session。
func (m *Model) Runtime() *launcher.Runtime {
	return m.rt
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.ti.Focus(),
		textinput.Blink,
	)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editingConfig {
		return m.updateConfigScreen(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ti.Width = max(10, msg.Width-4)
		if len(m.turns) == 0 {
			m.setViewportMain(m.placeholder())
		} else {
			m.setViewportMain(m.transcript())
		}
		return m, nil

	case tea.KeyMsg:
		if m.busy {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.cancelRun()
				return m, nil
			default:
				return m, nil
			}
		}
		if msg.String() == "ctrl+c" {
			return m.copyToClipboard()
		}
		if msg.String() == "tab" {
			if m.applySlashTabCompletion() {
				return m, textinput.Blink
			}
		}
		switch msg.String() {
		case "pgup", "pgdown", "ctrl+u", "ctrl+d":
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
		if msg.String() == "enter" {
			return m.submitInput()
		}
		if msg.String() == "esc" {
			return m, tea.Quit
		}

	case streamMsg:
		if msg.runID != m.runID {
			return m, waitStream(m.ch)
		}
		if msg.err != nil {
			launcher.MergeSessionValues(m.rt.Session, msg.session)
			if m.pendingReasoning.Len() > 0 {
				m.appendAssistantReasoning(m.pendingReasoning.String())
				m.pendingReasoning.Reset()
			}
			if m.pending.Len() > 0 {
				m.appendAssistantChunk(m.pending.String())
				m.pending.Reset()
			}
			m.finalizeAssistantStream()
			m.stripTrailingEmptyAssistant()
			m.turns = append(m.turns, turn{role: "error", text: msg.err.Error()})
			m.finishRun()
			_ = m.saveConversation()
			m.setViewportMain(m.transcript())
			return m, m.ti.Focus()
		}
		if msg.done {
			launcher.MergeSessionValues(m.rt.Session, msg.session)
			if m.pendingReasoning.Len() > 0 {
				m.appendAssistantReasoning(m.pendingReasoning.String())
				m.pendingReasoning.Reset()
			}
			if m.pending.Len() > 0 {
				m.appendAssistantChunk(m.pending.String())
				m.pending.Reset()
			}
			m.finalizeAssistantStream()
			if msg.cancelled {
				m.turns = append(m.turns, turn{role: "meta", text: "已取消"})
			}
			m.finishRun()
			_ = m.saveConversation()
			m.setViewportMain(m.transcript())
			return m, m.ti.Focus()
		}
		if len(msg.toolCalls) > 0 {
			if m.pendingReasoning.Len() > 0 {
				m.appendAssistantReasoning(m.pendingReasoning.String())
				m.pendingReasoning.Reset()
			}
			if m.pending.Len() > 0 {
				m.appendAssistantChunk(m.pending.String())
				m.pending.Reset()
			}
			m.finalizeAssistantStream()
			m.stripTrailingEmptyAssistant()
			for _, tc := range msg.toolCalls {
				m.turns = append(m.turns, turn{
					role:     "toolcall",
					toolName: tc.Function.Name,
					toolArgs: truncateRunes(tc.Function.Arguments, maxToolArgRunes),
					toolID:   tc.ID,
				})
			}
			m.setViewportMain(m.transcript())
			return m, waitStream(m.ch)
		}
		if msg.toolResultName != "" || msg.toolResultBody != "" {
			name := msg.toolResultName
			if name == "" {
				name = "tool"
			}
			m.turns = append(m.turns, turn{
				role:     "toolresult",
				toolName: name,
				text:     truncateRunes(msg.toolResultBody, maxToolBodyRunes),
			})
			m.setViewportMain(m.transcript())
			return m, waitStream(m.ch)
		}
		if msg.reasoning != "" {
			m.ensureStreamingAssistantTurn()
			m.pendingReasoning.WriteString(msg.reasoning)
			m.setViewportMain(m.transcriptStreaming())
			return m, waitStream(m.ch)
		}
		if msg.text != "" {
			m.ensureStreamingAssistantTurn()
			m.pending.WriteString(msg.text)
			m.setViewportMain(m.transcriptStreaming())
			return m, waitStream(m.ch)
		}
		return m, waitStream(m.ch)
	}

	if _, ok := msg.(tea.MouseMsg); ok {
		var vcmd tea.Cmd
		m.vp, vcmd = m.vp.Update(msg)
		var cmd tea.Cmd
		m.ti, cmd = m.ti.Update(msg)
		return m, tea.Batch(vcmd, cmd)
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m *Model) cancelRun() {
	if m.runCancel != nil {
		m.runCancel()
	}
}

func (m *Model) finishRun() {
	m.busy = false
	m.runCancel = nil
	m.pending.Reset()
	m.pendingReasoning.Reset()
}

func (m *Model) submitInput() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.ti.Value())
	if raw == "" {
		return m, nil
	}
	if strings.HasPrefix(raw, "/") {
		fields := strings.Fields(raw)
		if len(fields) > 0 && strings.EqualFold(fields[0], "/agent") {
			m.ti.Reset()
			return m.handleAgentCommand(strings.TrimSpace(raw))
		}
		first := strings.ToLower(strings.TrimSpace(strings.SplitN(raw, " ", 2)[0]))
		if first == "/config" {
			m.ti.Reset()
			return m.openConfigEditor()
		}
		if first == "/new" {
			m.ti.Reset()
			return m.startNewConversation()
		}
	}
	m.ti.Reset()
	return m.startRun(raw)
}

func (m *Model) handleAgentCommand(raw string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(raw)
	if len(fields) < 3 || !strings.EqualFold(fields[1], "mode") {
		m.turns = append(m.turns, turn{role: "meta", text: "用法: /agent mode <react|deep|sequential|parallel|loop|supervisor|plan_execute|custom>"})
		m.setViewportMain(m.transcript())
		return m, m.ti.Focus()
	}
	mode := agentconfig.AgentMode(strings.ToLower(fields[2]))
	if err := agentconfig.TryValidateModeSwitch(m.rt.Root, mode); err != nil {
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("无法切换: %v", err)})
		m.setViewportMain(m.transcript())
		return m, m.ti.Focus()
	}
	if err := agentconfig.PatchAgentModeInYAMLFile(m.cfgPath, mode); err != nil {
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("写入配置: %v", err)})
		m.setViewportMain(m.transcript())
		return m, m.ti.Focus()
	}
	ctx := context.Background()
	rt, err := launcher.Load(ctx, m.cfgPath)
	if err != nil {
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("重新加载: %v", err)})
		m.setViewportMain(m.transcript())
		return m, m.ti.Focus()
	}
	logPath, lerr := brookdir.LogFile()
	if lerr != nil {
		m.turns = append(m.turns, turn{role: "meta", text: fmt.Sprintf("已切换 mode=%s（日志路径: %v）", mode, lerr)})
	} else if oerr := launcher.ApplyObservability(rt.Root, logPath, true); oerr != nil {
		m.turns = append(m.turns, turn{role: "meta", text: fmt.Sprintf("已切换 mode=%s（日志: %v）", mode, oerr)})
	} else {
		m.turns = append(m.turns, turn{role: "meta", text: fmt.Sprintf("已切换 agent.mode=%s 并写回 %s", mode, m.cfgPath)})
	}
	m.rt = rt
	m.agentName = rt.Root.Agent.Name
	m.refreshMetaLine()
	m.setViewportMain(m.transcript())
	return m, m.ti.Focus()
}

func (m *Model) startNewConversation() (tea.Model, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	id := uuid.New().String()
	path, err := brookdir.ConversationFile(id)
	if err != nil {
		m.turns = append(m.turns, turn{role: "error", text: err.Error()})
		m.setViewportMain(m.transcript())
		return m, m.ti.Focus()
	}
	m.convID = id
	m.convFilePath = path
	m.turns = []turn{{role: "meta", text: fmt.Sprintf("已开始新会话（%s…）", id[:8])}}
	_ = brookdir.WriteCurrentConversationID(id)
	m.refreshMetaLine()
	m.setViewportMain(m.transcript())
	return m, m.ti.Focus()
}

func (m *Model) openConfigEditor() (tea.Model, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	b, err := os.ReadFile(m.cfgPath)
	if err != nil {
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("读取配置: %v", err)})
		m.setViewportMain(m.transcript())
		return m, nil
	}
	m.cfgTA.SetValue(string(b))
	m.editingConfig = true
	m.layoutConfigEditor()
	m.ti.Blur()
	_ = m.cfgTA.Focus()
	return m, textarea.Blink
}

func (m *Model) layoutConfigEditor() {
	if m.width == 0 {
		return
	}
	hh := lipgloss.Height(m.renderConfigHeader())
	fh := lipgloss.Height(m.renderConfigFooter())
	h := max(4, m.height-hh-fh-2)
	m.cfgTA.SetWidth(max(10, m.width-4))
	m.cfgTA.SetHeight(h)
}

func (m *Model) updateConfigScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutConfigEditor()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.editingConfig = false
			m.cfgTA.Blur()
			return m, m.ti.Focus()
		case "ctrl+c":
			return m.copyConfigEditorToClipboard()
		case "ctrl+q":
			return m, tea.Quit
		case "ctrl+s":
			return m.saveConfigAndReload()
		default:
		}
	}
	var cmd tea.Cmd
	m.cfgTA, cmd = m.cfgTA.Update(msg)
	return m, cmd
}

func (m *Model) saveConfigAndReload() (tea.Model, tea.Cmd) {
	body := m.cfgTA.Value()
	if _, err := agentconfig.LoadYAMLWithDir([]byte(body), filepath.Dir(m.cfgPath)); err != nil {
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("配置无效，未保存: %v", err)})
		m.setViewportMain(m.transcript())
		m.editingConfig = false
		m.cfgTA.Blur()
		return m, m.ti.Focus()
	}
	if err := os.WriteFile(m.cfgPath, []byte(body), 0o600); err != nil {
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("写入配置: %v", err)})
		m.setViewportMain(m.transcript())
		m.editingConfig = false
		m.cfgTA.Blur()
		return m, m.ti.Focus()
	}
	ctx := context.Background()
	rt, err := launcher.Load(ctx, m.cfgPath)
	if err != nil {
		m.turns = append(m.turns, turn{role: "error", text: fmt.Sprintf("重新加载: %v", err)})
		m.setViewportMain(m.transcript())
		m.editingConfig = false
		m.cfgTA.Blur()
		return m, m.ti.Focus()
	}
	doneMsg := "配置已保存并重新加载"
	logPath, lerr := brookdir.LogFile()
	if lerr != nil {
		doneMsg = fmt.Sprintf("配置已保存并重新加载（日志路径: %v）", lerr)
	} else if oerr := launcher.ApplyObservability(rt.Root, logPath, true); oerr != nil {
		doneMsg = fmt.Sprintf("配置已保存并重新加载（日志: %v）", oerr)
	}
	m.rt = rt
	m.agentName = rt.Root.Agent.Name
	m.refreshMetaLine()
	m.turns = append(m.turns, turn{role: "meta", text: doneMsg})
	m.setViewportMain(m.transcript())
	m.editingConfig = false
	m.cfgTA.Blur()
	return m, m.ti.Focus()
}

func (m *Model) ensureStreamingAssistantTurn() {
	if len(m.turns) == 0 {
		m.turns = append(m.turns, turn{role: "assistant", text: "", stream: true})
		return
	}
	last := m.turns[len(m.turns)-1]
	if last.role == "assistant" && last.stream {
		return
	}
	m.turns = append(m.turns, turn{role: "assistant", text: "", stream: true})
}

func (m *Model) finalizeAssistantStream() {
	if len(m.turns) == 0 {
		return
	}
	last := len(m.turns) - 1
	if m.turns[last].role != "assistant" || !m.turns[last].stream {
		return
	}
	m.turns[last].stream = false
}

func (m *Model) stripTrailingEmptyAssistant() {
	n := len(m.turns)
	if n == 0 {
		return
	}
	last := n - 1
	if m.turns[last].role != "assistant" {
		return
	}
	if strings.TrimSpace(m.turns[last].text+m.turns[last].reasoning+m.pending.String()+m.pendingReasoning.String()) != "" {
		return
	}
	m.turns = m.turns[:last]
}

func (m *Model) startRun(user string) (tea.Model, tea.Cmd) {
	m.runID++
	rid := m.runID
	m.turns = append(m.turns, turn{role: "user", text: user})
	m.turns = append(m.turns, turn{role: "assistant", text: "", stream: true})
	m.busy = true
	m.pending.Reset()
	m.pendingReasoning.Reset()
	m.ti.Blur()

	ctx, cancel := context.WithCancel(context.Background())
	m.runCancel = cancel

	m.setViewportMain(m.transcriptStreaming())

	r := m.rt.Runner
	sess := m.rt.Session
	cb, snapSession := launcher.SessionValuesSyncHandler()

	maxCtx := m.rt.Root.Memory.MaxContextMessages
	hist := conversation.TurnsToMessages(m.historyTurnsForRun(), maxCtx)
	runMsgs := make([]adk.Message, 0, len(hist)+1)
	runMsgs = append(runMsgs, hist...)
	runMsgs = append(runMsgs, schema.UserMessage(user))

	go func() {
		defer func() {
			done := streamMsg{runID: rid, done: true}
			done.cancelled = errors.Is(ctx.Err(), context.Canceled)
			done.session = snapSession()
			m.ch <- done
		}()

		iter := r.Run(ctx, runMsgs, adk.WithSessionValues(sess), adk.WithCallbacks(cb))
		for {
			if ctx.Err() != nil {
				return
			}
			ev, ok := iter.Next()
			if !ok {
				break
			}
			if ev == nil {
				continue
			}
			if ev.Err != nil {
				if errors.Is(ev.Err, context.Canceled) {
					return
				}
				m.ch <- streamMsg{runID: rid, err: ev.Err, session: snapSession()}
				return
			}
			if ev.Output == nil || ev.Output.MessageOutput == nil {
				continue
			}
			mv := ev.Output.MessageOutput
			if mv.Role == schema.Tool {
				if err := m.emitToolResult(rid, mv); err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					m.ch <- streamMsg{runID: rid, err: err, session: snapSession()}
					return
				}
				continue
			}
			if err := m.emitAssistantOutput(ctx, rid, mv); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				m.ch <- streamMsg{runID: rid, err: err, session: snapSession()}
				return
			}
		}
	}()

	return m, waitStream(m.ch)
}

func (m *Model) emitToolResult(rid int, mv *adk.MessageVariant) error {
	var full *schema.Message
	var err error
	if mv.IsStreaming && mv.MessageStream != nil {
		var chunks []*schema.Message
		for {
			msg, rerr := mv.MessageStream.Recv()
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return rerr
			}
			if msg != nil {
				chunks = append(chunks, msg)
			}
		}
		full, err = schema.ConcatMessages(chunks)
		if err != nil {
			return err
		}
	} else {
		full, err = mv.GetMessage()
		if err != nil {
			return err
		}
	}
	name := mv.ToolName
	if full != nil {
		if full.ToolName != "" {
			name = full.ToolName
		}
		body := full.Content
		m.ch <- streamMsg{runID: rid, toolResultName: name, toolResultBody: body}
	}
	return nil
}

func (m *Model) emitAssistantOutput(ctx context.Context, rid int, mv *adk.MessageVariant) error {
	if mv.IsStreaming && mv.MessageStream != nil {
		var chunks []*schema.Message
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			msg, err := mv.MessageStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if msg != nil {
				chunks = append(chunks, msg)
				if msg.ReasoningContent != "" {
					m.ch <- streamMsg{runID: rid, reasoning: msg.ReasoningContent}
				}
				if msg.Content != "" {
					m.ch <- streamMsg{runID: rid, text: msg.Content}
				}
			}
		}
		if len(chunks) == 0 {
			return nil
		}
		full, err := schema.ConcatMessages(chunks)
		if err != nil {
			return err
		}
		if full != nil && len(full.ToolCalls) > 0 {
			m.ch <- streamMsg{runID: rid, toolCalls: full.ToolCalls}
		}
		return nil
	}
	if mv.Message != nil {
		msg := mv.Message
		if msg.ReasoningContent != "" {
			m.ch <- streamMsg{runID: rid, reasoning: msg.ReasoningContent}
		}
		if msg.Content != "" {
			m.ch <- streamMsg{runID: rid, text: msg.Content}
		}
		if len(msg.ToolCalls) > 0 {
			m.ch <- streamMsg{runID: rid, toolCalls: msg.ToolCalls}
		}
	}
	return nil
}

func waitStream(ch <-chan streamMsg) tea.Cmd {
	return func() tea.Msg {
		v, ok := <-ch
		if !ok {
			return streamMsg{done: true}
		}
		return v
	}
}

func (m *Model) appendAssistantChunk(s string) {
	if len(m.turns) == 0 {
		return
	}
	last := len(m.turns) - 1
	if m.turns[last].role != "assistant" {
		return
	}
	m.turns[last].text += s
	m.turns[last].stream = false
}

func (m *Model) appendAssistantReasoning(s string) {
	if len(m.turns) == 0 {
		return
	}
	last := len(m.turns) - 1
	if m.turns[last].role != "assistant" {
		return
	}
	m.turns[last].reasoning += s
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.editingConfig {
		return styleApp.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				m.renderConfigHeader(),
				m.cfgTA.View(),
				m.renderConfigFooter(),
			),
		)
	}
	body := m.vp.View()
	return styleApp.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.renderHeader(),
			body,
			m.renderFooter(),
			m.renderInputBlock(),
		),
	)
}

func (m *Model) renderInputBlock() string {
	ruleW := max(0, m.width-2)
	rule := styleInputRule.Render(strings.Repeat("─", ruleW))
	return lipgloss.JoinVertical(lipgloss.Left, rule, m.ti.View(), rule)
}

func (m *Model) renderConfigHeader() string {
	left := styleTitle.Render("Brook")
	right := styleMeta.Render(m.cfgPath)
	line := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ·  ", styleBadge.Render("编辑配置"), "  ·  ", right)
	return styleHeader.Width(m.width - 2).Render(line)
}

func (m *Model) renderConfigFooter() string {
	hint := "ctrl+s 保存 · esc 返回 · ctrl+c 复制 · ctrl+v 粘贴 · ctrl+q 退出"
	return styleFooter.Width(m.width - 2).Render(hint)
}

func (m *Model) renderHeader() string {
	left := styleTitle.Render("Brook")
	badge := styleBadge.Render(m.agentName)
	right := styleMeta.Render(m.metaLine)
	line := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", badge, "  ·  ", right)
	return styleHeader.Width(m.width - 2).Render(line)
}

func (m *Model) placeholder() string {
	return styleMeta.Render("  Enter 发送 · /agent mode · /config · /new · Tab 补全 · pgup/pgdn · Ctrl+C/V · Esc 退出")
}

func (m *Model) renderFooter() string {
	hint := "enter · /agent mode · /config · /new · tab · 滚轮/pgup · ctrl+c/v · esc 退出"
	if m.busy {
		hint = "生成中… · esc / ctrl+c 取消"
	}
	return styleFooter.Width(m.width - 2).Render(hint)
}

// syncViewportLayout 按实际渲染高度计算 viewport，避免输入区（含上下分隔线）高度被低估导致整屏错位。
// busy 切换会改变页脚文案高度，须在刷新内容前调用，使主区域与 header/footer/input 之和等于终端高度。
func (m *Model) syncViewportLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	headerH := lipgloss.Height(m.renderHeader())
	footerH := lipgloss.Height(m.renderFooter())
	inputH := lipgloss.Height(m.renderInputBlock())
	vpH := m.height - headerH - footerH - inputH
	if vpH < 4 {
		vpH = 4
	}
	m.vp.Width = m.width - 2
	m.vp.Height = vpH
}

func (m *Model) setViewportMain(content string) {
	m.syncViewportLayout()
	m.vp.SetContent(content)
	m.vp.GotoBottom()
}

func (m *Model) transcript() string {
	var b strings.Builder
	for _, t := range m.turns {
		b.WriteString(m.renderTurn(t))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Model) transcriptStreaming() string {
	var b strings.Builder
	for i, t := range m.turns {
		isLast := i == len(m.turns)-1
		if isLast && t.role == "assistant" && t.stream {
			tt := t
			tt.text += m.pending.String()
			tt.reasoning += m.pendingReasoning.String()
			b.WriteString(m.renderTurn(tt))
		} else {
			b.WriteString(m.renderTurn(t))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Model) renderTurn(t turn) string {
	w := max(20, m.width-6)
	switch t.role {
	case "user":
		label := styleUserLabel.Render("You")
		body := styleUserText.Width(w).Render(t.text)
		return lipgloss.JoinVertical(lipgloss.Left, label, body)
	case "assistant":
		label := styleAsstLabel.Render("Brook")
		var stack []string
		if strings.TrimSpace(t.reasoning) != "" {
			stack = append(stack, lipgloss.JoinVertical(lipgloss.Left,
				styleTranscriptHint.Render("thinking"),
				styleReasoning.Width(w).Render(strings.TrimRight(t.reasoning, "\n")),
			))
		}
		if strings.TrimSpace(t.text) != "" {
			stack = append(stack, styleAsstBody.Width(w).Render(t.text))
		} else if len(stack) == 0 {
			stack = append(stack, styleAsstBody.Width(w).Render("…"))
		}
		return lipgloss.JoinVertical(lipgloss.Left, label, lipgloss.JoinVertical(lipgloss.Left, stack...))
	case "toolcall":
		title := styleToolHeader.Render("⏵ " + t.toolName)
		sub := styleMeta.Render(t.toolID)
		args := styleToolArgs.Width(w).Render(t.toolArgs)
		return lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", sub), args)
	case "toolresult":
		lbl := styleToolResultLabel.Render("tool · " + t.toolName)
		body := styleToolResultBody.Width(w).Render(t.text)
		return lipgloss.JoinVertical(lipgloss.Left, lbl, body)
	case "meta":
		return styleMetaNote.Render(t.text)
	case "error":
		return styleErr.Width(max(20, m.width-6)).Render(t.text)
	default:
		return styleErr.Render(t.text)
	}
}
