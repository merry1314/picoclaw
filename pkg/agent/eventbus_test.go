package agent

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestEventBus_SubscribeEmitUnsubscribeClose(t *testing.T) {
	eventBus := NewEventBus()
	sub := eventBus.Subscribe(1)

	eventBus.Emit(Event{
		Kind: EventKindTurnStart,
		Meta: EventMeta{TurnID: "turn-1"},
	})

	select {
	case evt := <-sub.C:
		if evt.Kind != EventKindTurnStart {
			t.Fatalf("expected %v, got %v", EventKindTurnStart, evt.Kind)
		}
		if evt.Meta.TurnID != "turn-1" {
			t.Fatalf("expected turn id turn-1, got %q", evt.Meta.TurnID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	eventBus.Unsubscribe(sub.ID)
	if _, ok := <-sub.C; ok {
		t.Fatal("expected subscriber channel to be closed after unsubscribe")
	}

	eventBus.Close()
	closedSub := eventBus.Subscribe(1)
	if _, ok := <-closedSub.C; ok {
		t.Fatal("expected closed bus to return a closed subscriber channel")
	}
}

func TestEventBus_DropsWhenSubscriberIsFull(t *testing.T) {
	eventBus := NewEventBus()
	sub := eventBus.Subscribe(1)
	defer eventBus.Unsubscribe(sub.ID)

	start := time.Now()
	for i := 0; i < 1000; i++ {
		eventBus.Emit(Event{Kind: EventKindLLMRequest})
	}

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Emit took too long with a blocked subscriber: %s", elapsed)
	}

	if got := eventBus.Dropped(EventKindLLMRequest); got != 999 {
		t.Fatalf("expected 999 dropped events, got %d", got)
	}
}

type scriptedToolProvider struct {
	calls int
}

func (m *scriptedToolProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{
					ID:        "call-1",
					Name:      "mock_custom",
					Arguments: map[string]any{"task": "ping"},
				},
			},
		}, nil
	}

	return &providers.LLMResponse{
		Content: "done",
	}, nil
}

func (m *scriptedToolProvider) GetDefaultModel() string {
	return "scripted-tool-model"
}

func TestAgentLoop_EmitsMinimalTurnEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-eventbus-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &scriptedToolProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	al.RegisterTool(&mockCustomTool{})
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	response, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		SessionKey:      "session-1",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "run tool",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if response != "done" {
		t.Fatalf("expected final response 'done', got %q", response)
	}

	events := collectEventStream(sub.C)
	if len(events) != 8 {
		t.Fatalf("expected 8 events, got %d", len(events))
	}

	kinds := make([]EventKind, 0, len(events))
	for _, evt := range events {
		kinds = append(kinds, evt.Kind)
	}

	expectedKinds := []EventKind{
		EventKindTurnStart,
		EventKindLLMRequest,
		EventKindLLMResponse,
		EventKindToolExecStart,
		EventKindToolExecEnd,
		EventKindLLMRequest,
		EventKindLLMResponse,
		EventKindTurnEnd,
	}
	if !slices.Equal(kinds, expectedKinds) {
		t.Fatalf("unexpected event sequence: got %v want %v", kinds, expectedKinds)
	}

	turnID := events[0].Meta.TurnID
	for i, evt := range events {
		if evt.Meta.TurnID != turnID {
			t.Fatalf("event %d has mismatched turn id %q, want %q", i, evt.Meta.TurnID, turnID)
		}
		if evt.Meta.SessionKey != "session-1" {
			t.Fatalf("event %d has session key %q, want session-1", i, evt.Meta.SessionKey)
		}
	}

	startPayload, ok := events[0].Payload.(TurnStartPayload)
	if !ok {
		t.Fatalf("expected TurnStartPayload, got %T", events[0].Payload)
	}
	if startPayload.UserMessage != "run tool" {
		t.Fatalf("expected user message 'run tool', got %q", startPayload.UserMessage)
	}

	toolStartPayload, ok := events[3].Payload.(ToolExecStartPayload)
	if !ok {
		t.Fatalf("expected ToolExecStartPayload, got %T", events[3].Payload)
	}
	if toolStartPayload.Tool != "mock_custom" {
		t.Fatalf("expected tool name mock_custom, got %q", toolStartPayload.Tool)
	}

	toolEndPayload, ok := events[4].Payload.(ToolExecEndPayload)
	if !ok {
		t.Fatalf("expected ToolExecEndPayload, got %T", events[4].Payload)
	}
	if toolEndPayload.Tool != "mock_custom" {
		t.Fatalf("expected tool end payload for mock_custom, got %q", toolEndPayload.Tool)
	}
	if toolEndPayload.IsError {
		t.Fatal("expected mock_custom tool to succeed")
	}

	turnEndPayload, ok := events[len(events)-1].Payload.(TurnEndPayload)
	if !ok {
		t.Fatalf("expected TurnEndPayload, got %T", events[len(events)-1].Payload)
	}
	if turnEndPayload.Status != TurnEndStatusCompleted {
		t.Fatalf("expected completed turn, got %q", turnEndPayload.Status)
	}
	if turnEndPayload.Iterations != 2 {
		t.Fatalf("expected 2 iterations, got %d", turnEndPayload.Iterations)
	}
}

func collectEventStream(ch <-chan Event) []Event {
	var events []Event
	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, evt)
		default:
			return events
		}
	}
}

var _ tools.Tool = (*mockCustomTool)(nil)
