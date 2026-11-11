package executor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/MoeForge/moego/ojson"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
)

func TestClaudeParameterValidationStopsBeforeUpstream(t *testing.T) {
	for _, surface := range []string{"execute", "stream", "count", "count baseline"} {
		t.Run(surface, func(t *testing.T) {
			attempts := 0
			transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				attempts++
				return nil, errors.New("unexpected upstream call")
			})
			ctx := context.WithValue(t.Context(), "cliproxy.roundtripper", http.RoundTripper(transport))
			cfg := &config.Config{}
			if surface != "count baseline" {
				cfg.ClaudeHeaderDefaults.AnthropicBetaStrip = []string{claudeThinkingDisplayUpdatesBeta}
			}
			executor := NewClaudeExecutor(cfg)
			auth := &cliproxyauth.Auth{Attributes: map[string]string{"api_key": "test-parameter-key", "cloak_mode": "always"}}
			request := cliproxyexecutor.Request{Model: "claude-sonnet-5-5", Payload: []byte(`{"model":"claude-sonnet-5-5","max_tokens":16,"thinking":{"type":"adaptive","display":"updates"},"messages":[{"role":"user","content":"hello"}]}`)}
			options := cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude, Headers: http.Header{"Anthropic-Beta": []string{claudeThinkingDisplayUpdatesBeta}}}
			var err error
			switch surface {
			case "execute":
				_, err = executor.Execute(ctx, auth, request, options)
			case "stream":
				_, err = executor.ExecuteStream(ctx, auth, request, options)
			case "count", "count baseline":
				_, err = executor.CountTokens(ctx, auth, request, options)
			}
			if attempts != 0 {
				t.Fatalf("upstream calls = %d, want zero", attempts)
			}
			var parameterError helps.ClaudeParameterError
			if !errors.As(err, &parameterError) || parameterError.StatusCode() != 400 {
				t.Fatalf("error = %v, want parameter 400", err)
			}
			if got := ojson.GetBytes(parameterError.ResponseBody(), "error.code").String(); got != "unsupported_request_parameter" {
				t.Fatalf("code = %q", got)
			}
			if parameterError.Param != "thinking.display" {
				t.Fatalf("param = %q", parameterError.Param)
			}
		})
	}
}

func TestClaudeParameterErrorStopsCredentialRotation(t *testing.T) {
	for _, surface := range []string{"execute", "stream", "count"} {
		t.Run(surface, func(t *testing.T) {
			attempts := 0
			transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				attempts++
				return &http.Response{StatusCode: 400, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"thinking.adaptive.display: Input should be 'summarized', 'omitted'"}}`)), Request: req}, nil
			})
			ctx := context.WithValue(t.Context(), "cliproxy.roundtripper", http.RoundTripper(transport))
			manager := cliproxyauth.NewManager(nil, nil, nil)
			manager.SetRetryConfig(3, time.Second, 5)
			manager.RegisterExecutor(NewClaudeExecutor(&config.Config{}))
			for _, id := range []string{"parameter-one-" + surface, "parameter-two-" + surface} {
				registry.GetGlobalRegistry().RegisterClient(id, "claude", []*registry.ModelInfo{{ID: "claude-sonnet-5-5"}})
				t.Cleanup(func() { registry.GetGlobalRegistry().UnregisterClient(id) })
				_, err := manager.Register(ctx, &cliproxyauth.Auth{ID: id, Provider: "claude", Status: cliproxyauth.StatusActive, Metadata: map[string]any{"request_scoped_errors": []config.RequestScopedErrorRule{{Status: 400, Match: []string{"unsupported_request_parameter"}, Action: "continue"}}}, Attributes: map[string]string{"api_key": "test-parameter-key", "cloak_mode": "never"}})
				if err != nil {
					t.Fatal(err)
				}
			}
			request := cliproxyexecutor.Request{Model: "claude-sonnet-5-5", Payload: []byte(`{"model":"claude-sonnet-5-5","max_tokens":16,"thinking":{"type":"adaptive","display":"summarized"},"messages":[{"role":"user","content":"hello"}]}`)}
			options := cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude}
			var err error
			switch surface {
			case "execute":
				_, err = manager.Execute(ctx, []string{"claude"}, request, options)
			case "stream":
				_, err = manager.ExecuteStream(ctx, []string{"claude"}, request, options)
			case "count":
				_, err = manager.ExecuteCount(ctx, []string{"claude"}, request, options)
			}
			if err == nil || attempts != 1 {
				t.Fatalf("error = %v, upstream calls = %d, want one", err, attempts)
			}
			if !strings.Contains(err.Error(), "unsupported_request_parameter") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestClaudeParameterValidationAllowsUpdatesWithFinalBeta(t *testing.T) {
	for _, surface := range []string{"execute", "stream", "count"} {
		t.Run(surface, func(t *testing.T) {
			attempts := 0
			transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				attempts++
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				if ojson.GetBytes(body, "thinking.display").String() != "updates" {
					t.Fatal("valid display was changed")
				}
				if err := helps.ValidateClaudeThinkingDisplay(body, req.Header); err != nil {
					t.Fatal(err)
				}
				return &http.Response{StatusCode: 400, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"unclassified channel error"}}`)), Request: req}, nil
			})
			ctx := context.WithValue(t.Context(), "cliproxy.roundtripper", http.RoundTripper(transport))
			cfg := &config.Config{}
			cfg.ClaudeHeaderDefaults.AnthropicBetaAppend = []string{claudeThinkingDisplayUpdatesBeta}
			executor := NewClaudeExecutor(cfg)
			auth := &cliproxyauth.Auth{Attributes: map[string]string{"api_key": "test-parameter-key", "cloak_mode": "always"}}
			request := cliproxyexecutor.Request{Model: "claude-sonnet-5-5", Payload: []byte(`{"model":"claude-sonnet-5-5","max_tokens":16,"thinking":{"type":"adaptive","display":"updates"},"messages":[{"role":"user","content":"hello"}]}`)}
			options := cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude}
			var err error
			switch surface {
			case "execute":
				_, err = executor.Execute(ctx, auth, request, options)
			case "stream":
				_, err = executor.ExecuteStream(ctx, auth, request, options)
			case "count":
				_, err = executor.CountTokens(ctx, auth, request, options)
			}
			var parameterError helps.ClaudeParameterError
			if attempts != 1 || errors.As(err, &parameterError) {
				t.Fatalf("calls = %d, error = %v", attempts, err)
			}
		})
	}
}
