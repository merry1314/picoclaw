package session

import "testing"

func TestIsExplicitSessionKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"sk_v1_abc", true},
		{"agent:main:direct:user123", true},
		{"custom-key", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsExplicitSessionKey(tt.key); got != tt.want {
			t.Fatalf("IsExplicitSessionKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestParseLegacyAgentSessionKey(t *testing.T) {
	parsed := ParseLegacyAgentSessionKey("agent:sales:telegram:direct:user123")
	if parsed == nil {
		t.Fatal("expected parsed legacy key, got nil")
	}
	if parsed.AgentID != "sales" {
		t.Fatalf("AgentID = %q, want sales", parsed.AgentID)
	}
	if parsed.Rest != "telegram:direct:user123" {
		t.Fatalf("Rest = %q, want telegram:direct:user123", parsed.Rest)
	}

	if got := ParseLegacyAgentSessionKey("sk_v1_abc"); got != nil {
		t.Fatalf("expected nil for opaque key, got %+v", got)
	}
}

func TestBuildLegacyDirectAliases(t *testing.T) {
	aliases := BuildLegacyDirectAliases("Main", "Telegram", "BotA", "User123")
	want := []string{
		"agent:main:direct:user123",
		"agent:main:telegram:direct:user123",
		"agent:main:telegram:bota:direct:user123",
	}
	if len(aliases) != len(want) {
		t.Fatalf("len(aliases) = %d, want %d", len(aliases), len(want))
	}
	for i := range want {
		if aliases[i] != want[i] {
			t.Fatalf("aliases[%d] = %q, want %q", i, aliases[i], want[i])
		}
	}
}

func TestBuildLegacyPeerAlias(t *testing.T) {
	got := BuildLegacyPeerAlias("Main", "Slack", "channel", "C001")
	if got != "agent:main:slack:channel:c001" {
		t.Fatalf("BuildLegacyPeerAlias() = %q", got)
	}
}

func TestBuildMainSessionKey(t *testing.T) {
	got := BuildMainSessionKey("Main")
	if !IsOpaqueSessionKey(got) {
		t.Fatalf("BuildMainSessionKey() = %q, want opaque key", got)
	}
	if got != BuildOpaqueSessionKey("agent:main:main") {
		t.Fatalf("BuildMainSessionKey() = %q, want stable main-key hash", got)
	}
}
