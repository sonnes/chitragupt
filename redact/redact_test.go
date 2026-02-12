package redact

import (
	"testing"
	"time"

	"github.com/sonnes/chitragupt/core"
)

func TestAWSKeyDetection(t *testing.T) {
	rules := SecretRules()
	var r Rule
	for _, rule := range rules {
		if rule.Name() == "aws_key" {
			r = rule
			break
		}
	}
	if r == nil {
		t.Fatal("aws_key rule not found")
	}

	matches := r.Detect("export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Value != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("expected AKIAIOSFODNN7EXAMPLE, got %s", matches[0].Value)
	}
	if rep := r.Replacement(matches[0]); rep != "[REDACTED:aws_key]" {
		t.Errorf("expected [REDACTED:aws_key], got %s", rep)
	}
}

func TestAPIKeyDetection(t *testing.T) {
	rules := SecretRules()
	var r Rule
	for _, rule := range rules {
		if rule.Name() == "api_key" {
			r = rule
			break
		}
	}
	if r == nil {
		t.Fatal("api_key rule not found")
	}

	tests := []struct {
		input string
		want  string
	}{
		{"sk-" + "abcdefghijklmnopqrstuvwxyz123456", "sk-abcdefghijklmnopqrstuvwxyz123456"},
		{"ghp_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij0123", "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij0123"},
	}
	for _, tt := range tests {
		matches := r.Detect(tt.input)
		if len(matches) != 1 {
			t.Errorf("input %q: expected 1 match, got %d", tt.input, len(matches))
			continue
		}
		if matches[0].Value != tt.want {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.want, matches[0].Value)
		}
	}
}

func TestPrivateKeyDetection(t *testing.T) {
	rules := SecretRules()
	var r Rule
	for _, rule := range rules {
		if rule.Name() == "private_key" {
			r = rule
			break
		}
	}
	if r == nil {
		t.Fatal("private_key rule not found")
	}

	input := "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAK..."
	matches := r.Detect(input)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if rep := r.Replacement(matches[0]); rep != "[REDACTED:private_key]" {
		t.Errorf("expected [REDACTED:private_key], got %s", rep)
	}
}

func TestConnectionStringDetection(t *testing.T) {
	rules := SecretRules()
	var r Rule
	for _, rule := range rules {
		if rule.Name() == "connection_string" {
			r = rule
			break
		}
	}
	if r == nil {
		t.Fatal("connection_string rule not found")
	}

	input := `DATABASE_URL=postgres://admin:s3cret@db.example.com:5432/mydb`
	matches := r.Detect(input)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if rep := r.Replacement(matches[0]); rep != "[REDACTED:connection_string]" {
		t.Errorf("expected [REDACTED:connection_string], got %s", rep)
	}
}

func TestJWTDetection(t *testing.T) {
	rules := SecretRules()
	var r Rule
	for _, rule := range rules {
		if rule.Name() == "jwt" {
			r = rule
			break
		}
	}
	if r == nil {
		t.Fatal("jwt rule not found")
	}

	input := "Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	matches := r.Detect(input)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
}

func TestEmailDetection(t *testing.T) {
	rules := PIIRules()
	var r Rule
	for _, rule := range rules {
		if rule.Name() == "email" {
			r = rule
			break
		}
	}
	if r == nil {
		t.Fatal("email rule not found")
	}

	matches := r.Detect("contact user@example.com for help")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Value != "user@example.com" {
		t.Errorf("expected user@example.com, got %s", matches[0].Value)
	}
}

func TestIPv4Detection(t *testing.T) {
	rules := PIIRules()
	var r Rule
	for _, rule := range rules {
		if rule.Name() == "ipv4" {
			r = rule
			break
		}
	}
	if r == nil {
		t.Fatal("ipv4 rule not found")
	}

	matches := r.Detect("server at 192.168.1.100 on port 8080")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Value != "192.168.1.100" {
		t.Errorf("expected 192.168.1.100, got %s", matches[0].Value)
	}
}

func TestPhoneDetection(t *testing.T) {
	rules := PIIRules()
	var r Rule
	for _, rule := range rules {
		if rule.Name() == "phone" {
			r = rule
			break
		}
	}
	if r == nil {
		t.Fatal("phone rule not found")
	}

	tests := []string{
		"call +1-555-123-4567",
		"call (555) 123-4567",
		"call 555-123-4567",
	}
	for _, input := range tests {
		matches := r.Detect(input)
		if len(matches) != 1 {
			t.Errorf("input %q: expected 1 match, got %d", input, len(matches))
		}
	}
}

func TestWalkAnyStrings(t *testing.T) {
	upper := func(s string) string { return s + "!" }

	tests := []struct {
		name  string
		input any
		want  any
	}{
		{"string", "hello", "hello!"},
		{"int", 42, 42},
		{"nil", nil, nil},
		{"map", map[string]any{"a": "b", "c": 1}, map[string]any{"a": "b!", "c": 1}},
		{"slice", []any{"x", 2, "y"}, []any{"x!", 2, "y!"}},
		{"nested", map[string]any{
			"cmd": "echo secret",
			"args": []any{"--flag", "value"},
		}, map[string]any{
			"cmd": "echo secret!",
			"args": []any{"--flag!", "value!"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := walkAny(tt.input, upper)
			if !anyEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWalkAnyMaxDepth(t *testing.T) {
	var v any = "leaf"
	for i := 0; i < maxWalkDepth+5; i++ {
		v = map[string]any{"nested": v}
	}

	called := false
	walkAny(v, func(s string) string {
		called = true
		return s
	})

	if called {
		t.Error("expected fn not to be called on deeply nested string")
	}
}

func TestRedactorTransform(t *testing.T) {
	now := time.Now()
	transcript := &core.Transcript{
		SessionID: "test-session",
		Agent:     "claude",
		CreatedAt: now,
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Text: "My key is AKIAIOSFODNN7EXAMPLE"},
				},
			},
			{
				Role: core.RoleAssistant,
				Content: []core.ContentBlock{
					{Type: core.BlockThinking, Text: "The user shared key AKIAIOSFODNN7EXAMPLE"},
					{Type: core.BlockText, Text: "I see your AWS key. Contact admin@corp.com for rotation."},
					{
						Type: core.BlockToolUse,
						Name: "Bash",
						Input: map[string]any{
							"command": "curl -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U' https://api.example.com",
						},
					},
				},
			},
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, Content: "connected to postgres://user:pass@db.internal:5432/prod"},
				},
			},
		},
	}

	r := New(Config{Secrets: true, PII: true})
	if err := r.Transform(transcript); err != nil {
		t.Fatal(err)
	}

	text := transcript.Messages[0].Content[0].Text
	if text != "My key is [REDACTED:aws_key]" {
		t.Errorf("text block: got %q", text)
	}

	thinking := transcript.Messages[1].Content[0].Text
	if thinking != "The user shared key [REDACTED:aws_key]" {
		t.Errorf("thinking block: got %q", thinking)
	}

	assistantText := transcript.Messages[1].Content[1].Text
	if assistantText != "I see your AWS key. Contact [REDACTED:email] for rotation." {
		t.Errorf("assistant text: got %q", assistantText)
	}

	input := transcript.Messages[1].Content[2].Input.(map[string]any)
	cmd := input["command"].(string)
	if cmd == "curl -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U' https://api.example.com" {
		t.Error("tool_use input JWT was not redacted")
	}

	result := transcript.Messages[2].Content[0].Content
	if result == "connected to postgres://user:pass@db.internal:5432/prod" {
		t.Error("tool_result connection string was not redacted")
	}
}

func TestRedactorSecretsOnly(t *testing.T) {
	transcript := &core.Transcript{
		SessionID: "test",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Text: "AKIAIOSFODNN7EXAMPLE and user@example.com"},
				},
			},
		},
	}

	r := New(Config{Secrets: true, PII: false})
	if err := r.Transform(transcript); err != nil {
		t.Fatal(err)
	}

	text := transcript.Messages[0].Content[0].Text
	if text != "[REDACTED:aws_key] and user@example.com" {
		t.Errorf("secrets-only: got %q", text)
	}
}

func TestRedactorAllowlist(t *testing.T) {
	transcript := &core.Transcript{
		SessionID: "test",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Text: "key AKIAIOSFODNN7EXAMPLE is safe"},
				},
			},
		},
	}

	r := New(Config{
		Secrets:   true,
		PII:       true,
		Allowlist: []string{`AKIAIOSFODNN7EXAMPLE`},
	})
	if err := r.Transform(transcript); err != nil {
		t.Fatal(err)
	}

	text := transcript.Messages[0].Content[0].Text
	if text != "key AKIAIOSFODNN7EXAMPLE is safe" {
		t.Errorf("allowlist: got %q", text)
	}
}

func TestRedactorNoRules(t *testing.T) {
	transcript := &core.Transcript{
		SessionID: "test",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Text: "AKIAIOSFODNN7EXAMPLE"},
				},
			},
		},
	}

	r := New(Config{Secrets: false, PII: false})
	if err := r.Transform(transcript); err != nil {
		t.Fatal(err)
	}

	text := transcript.Messages[0].Content[0].Text
	if text != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("no-rules: got %q", text)
	}
}

// anyEqual is a deep-equality check for test assertions.
func anyEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !anyEqual(v, bv[k]) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !anyEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
