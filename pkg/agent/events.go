package agent

import (
	"fmt"
	"time"
)

// EventKind identifies a structured agent-loop event.
type EventKind uint8

const (
	// EventKindTurnStart is emitted when a turn begins processing.
	EventKindTurnStart EventKind = iota
	// EventKindTurnEnd is emitted when a turn finishes, successfully or with an error.
	EventKindTurnEnd
	// EventKindLLMRequest is emitted before a provider chat request is made.
	EventKindLLMRequest
	// EventKindLLMResponse is emitted after a provider chat response is received.
	EventKindLLMResponse
	// EventKindToolExecStart is emitted immediately before a tool executes.
	EventKindToolExecStart
	// EventKindToolExecEnd is emitted immediately after a tool finishes executing.
	EventKindToolExecEnd
	// EventKindError is emitted when a turn encounters an execution error.
	EventKindError

	eventKindCount
)

var eventKindNames = [...]string{
	"turn_start",
	"turn_end",
	"llm_request",
	"llm_response",
	"tool_exec_start",
	"tool_exec_end",
	"error",
}

// String returns the stable string form of an EventKind.
func (k EventKind) String() string {
	if k >= eventKindCount {
		return fmt.Sprintf("event_kind(%d)", k)
	}
	return eventKindNames[k]
}

// Event is the structured envelope broadcast by the agent EventBus.
type Event struct {
	Kind    EventKind
	Time    time.Time
	Meta    EventMeta
	Payload any
}

// EventMeta contains correlation fields shared by all agent-loop events.
type EventMeta struct {
	AgentID      string
	TurnID       string
	ParentTurnID string
	SessionKey   string
	Iteration    int
	TracePath    string
	Source       string
}

// TurnEndStatus describes the terminal state of a turn.
type TurnEndStatus string

const (
	// TurnEndStatusCompleted indicates the turn finished normally.
	TurnEndStatusCompleted TurnEndStatus = "completed"
	// TurnEndStatusError indicates the turn ended because of an error.
	TurnEndStatusError TurnEndStatus = "error"
)

// TurnStartPayload describes the start of a turn.
type TurnStartPayload struct {
	Channel     string
	ChatID      string
	UserMessage string
	MediaCount  int
}

// TurnEndPayload describes the completion of a turn.
type TurnEndPayload struct {
	Status          TurnEndStatus
	Iterations      int
	Duration        time.Duration
	FinalContentLen int
}

// LLMRequestPayload describes an outbound LLM request.
type LLMRequestPayload struct {
	Model         string
	MessagesCount int
	ToolsCount    int
	MaxTokens     int
	Temperature   float64
}

// LLMResponsePayload describes an inbound LLM response.
type LLMResponsePayload struct {
	ContentLen   int
	ToolCalls    int
	HasReasoning bool
}

// ToolExecStartPayload describes a tool execution request.
type ToolExecStartPayload struct {
	Tool      string
	Arguments map[string]any
}

// ToolExecEndPayload describes the outcome of a tool execution.
type ToolExecEndPayload struct {
	Tool       string
	Duration   time.Duration
	ForLLMLen  int
	ForUserLen int
	IsError    bool
	Async      bool
}

// ErrorPayload describes an execution error inside the agent loop.
type ErrorPayload struct {
	Stage   string
	Message string
}
