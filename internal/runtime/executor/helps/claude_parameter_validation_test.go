package helps

import (
	"net/http"
	"testing"
)

func TestValidateClaudeThinkingDisplay(t *testing.T) {
	for _, test := range []struct {
		name, thinking, beta string
		reject               bool
	}{
		{name: "summarized", thinking: `{"type":"adaptive","display":"summarized"}`},
		{name: "omitted", thinking: `{"type":"enabled","display":"omitted"}`},
		{name: "null", thinking: `{"type":"adaptive","display":null}`},
		{name: "valid beta", thinking: `{"type":"adaptive","display":"updates"}`, beta: "future-feature, thinking-display-updates-2026-08-18"},
		{name: "missing beta", thinking: `{"type":"adaptive","display":"updates"}`, reject: true},
		{name: "partial beta", thinking: `{"type":"adaptive","display":"updates"}`, beta: "not-thinking-display-updates-2026-08-18", reject: true},
		{name: "unknown display", thinking: `{"type":"adaptive","display":"unknown"}`, reject: true},
		{name: "wrong type", thinking: `{"type":"adaptive","display":42}`, reject: true},
		{name: "disabled", thinking: `{"type":"disabled","display":"omitted"}`, reject: true},
		{name: "between tools", thinking: `{"type":"between_tools","display":"summarized"}`, reject: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := []byte(`{"thinking":` + test.thinking + `}`)
			err := ValidateClaudeThinkingDisplay(body, http.Header{"anthropic-beta": []string{test.beta}})
			if (err != nil) != test.reject {
				t.Fatalf("error = %v, reject = %v", err, test.reject)
			}
			if string(body) != `{"thinking":`+test.thinking+`}` {
				t.Fatal("validator modified request")
			}
		})
	}
}

func TestClassifyClaudeParameterError(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		reject bool
	}{
		{"display", 400, `{"error":{"type":"invalid_request_error","message":"thinking.adaptive.display: Input should be 'summarized', 'omitted'"}}`, true},
		{"tool choice", 400, `{"error":{"type":"invalid_request_error","message":"tool_choice: Input should be a valid dictionary"}}`, true},
		{"extra field", 400, `{"error":{"type":"invalid_request_error","message":"messages.2.output_config: Extra inputs are not permitted"}}`, true},
		{"model support", 400, `{"error":{"type":"invalid_request_error","message":"model: Input should be an available model"}}`, false},
		{"unknown bad request", 400, `{"error":{"type":"invalid_request_error","message":"request failed"}}`, false},
		{"overload", 503, `{"error":{"type":"invalid_request_error","message":"thinking.adaptive.display: Input should be 'omitted'"}}`, false},
		{"non schema", 400, `{"error":{"type":"invalid_request_error","message":"thinking.adaptive.display: service unavailable"}}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ClassifyClaudeParameterError(test.status, []byte(test.body))
			if (err != nil) != test.reject {
				t.Fatalf("error = %v, reject = %v", err, test.reject)
			}
		})
	}
}
