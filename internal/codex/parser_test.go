package codex

import "testing"

func TestParseLine_UserMessage(t *testing.T) {
	line := []byte(`{"timestamp":"2026-04-01T10:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"## My request for Codex:\nfix the login bug"}}`)

	msg := ParseLine(line, 2)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "user" {
		t.Errorf("role = %q, want %q", msg.Role, "user")
	}
	if msg.Text != "fix the login bug" {
		t.Errorf("text = %q, want %q", msg.Text, "fix the login bug")
	}
	if msg.ID != "line-000002" {
		t.Errorf("id = %q, want %q", msg.ID, "line-000002")
	}
	if msg.Timestamp != "2026-04-01T10:00:01.000Z" {
		t.Errorf("timestamp = %q, want %q", msg.Timestamp, "2026-04-01T10:00:01.000Z")
	}
}

func TestParseLine_AgentMessage(t *testing.T) {
	line := []byte(`{"timestamp":"2026-04-01T10:00:03.000Z","type":"event_msg","payload":{"type":"agent_message","message":"I found the login bug in the auth middleware."}}`)

	msg := ParseLine(line, 5)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "assistant" {
		t.Errorf("role = %q, want %q", msg.Role, "assistant")
	}
	if msg.Text != "I found the login bug in the auth middleware." {
		t.Errorf("text = %q, want %q", msg.Text, "I found the login bug in the auth middleware.")
	}
	if msg.ID != "line-000005" {
		t.Errorf("id = %q, want %q", msg.ID, "line-000005")
	}
}

func TestParseLine_ResponseItemUserMessage(t *testing.T) {
	line := []byte(`{"timestamp":"2026-04-01T10:00:02.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"follow-up user prompt"}]}}`)

	msg := ParseLine(line, 4)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "user" {
		t.Errorf("role = %q, want %q", msg.Role, "user")
	}
	if msg.Text != "follow-up user prompt" {
		t.Errorf("text = %q, want %q", msg.Text, "follow-up user prompt")
	}
	if msg.Source != "response_item" {
		t.Errorf("source = %q, want %q", msg.Source, "response_item")
	}
}

func TestParseLine_ResponseItemAssistantMessage(t *testing.T) {
	line := []byte(`{"timestamp":"2026-04-01T10:00:03.000Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"assistant reply"}]}}`)

	msg := ParseLine(line, 5)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Role != "assistant" {
		t.Errorf("role = %q, want %q", msg.Role, "assistant")
	}
	if msg.Text != "assistant reply" {
		t.Errorf("text = %q, want %q", msg.Text, "assistant reply")
	}
}

func TestParseLine_ResponseItemDeveloperSkipped(t *testing.T) {
	line := []byte(`{"timestamp":"2026-04-01T10:00:02.000Z","type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"system instructions"}]}}`)

	msg := ParseLine(line, 4)
	if msg != nil {
		t.Errorf("expected nil for developer response_item, got text=%q", msg.Text)
	}
}

func TestParseLine_NonChatEventSkipped(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{
			name: "session_meta",
			line: `{"timestamp":"2026-04-01T10:00:00.000Z","type":"session_meta","payload":{"id":"11111111-1111-1111-1111-111111111111","timestamp":"2026-04-01T10:00:00.000Z","cwd":"/home/user/project","originator":"codex-cli","cli_version":"0.1.0","source":"cli","model_provider":"openai"}}`,
		},
		{
			name: "reasoning",
			line: `{"timestamp":"2026-04-01T10:00:04.000Z","type":"event_msg","payload":{"type":"agent_reasoning","text":"checking the auth flow"}}`,
		},
		{
			name: "exec_command_end",
			line: `{"timestamp":"2026-04-02T11:00:02.000Z","type":"event_msg","payload":{"type":"exec_command_end","parsed_cmd":"kubectl apply -f deploy.yaml","aggregated_output":"ok","exit_code":0}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := ParseLine([]byte(tc.line), 1)
			if msg != nil {
				t.Errorf("expected nil for %s, got text=%q", tc.name, msg.Text)
			}
		})
	}
}

func TestParseLine_MalformedJSON(t *testing.T) {
	msg := ParseLine([]byte(`{not valid json`), 3)
	if msg != nil {
		t.Error("expected nil for malformed JSON")
	}
}
