package helps

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/MoeForge/moego/ojson"
)

type ClaudeParameterError struct {
	Param   string
	Message string
}

func (e ClaudeParameterError) Error() string       { return string(e.ResponseBody()) }
func (ClaudeParameterError) StatusCode() int       { return http.StatusBadRequest }
func (ClaudeParameterError) IsRequestScoped() bool { return true }
func (ClaudeParameterError) IsRequestStop() bool   { return true }
func (ClaudeParameterError) DirectResponse() bool  { return true }
func (e ClaudeParameterError) ResponseBody() []byte {
	doc := ojson.NewDoc([]byte(`{"type":"error","error":{"type":"invalid_request_error","code":"unsupported_request_parameter"}}`))
	_ = doc.Set("error.param", e.Param)
	_ = doc.Set("error.message", e.Message)
	return doc.Bytes()
}

// ValidateClaudeThinkingDisplay checks the final payload and emitted feature headers.
func ValidateClaudeThinkingDisplay(body []byte, headers http.Header) error {
	display := ojson.GetBytesNoCopy(body, "thinking.display")
	if !display.Exists() || display.Type == ojson.Null {
		return nil
	}
	thinkingType := ojson.GetBytesNoCopy(body, "thinking.type").String()
	if thinkingType == "disabled" || thinkingType == "between_tools" {
		return ClaudeParameterError{Param: "thinking.display", Message: "thinking.display is not supported with thinking.type=" + thinkingType}
	}
	if display.Type == ojson.String {
		switch display.String() {
		case "summarized", "omitted":
			return nil
		case "updates":
			for _, value := range HeaderValuesCaseInsensitive(headers, "Anthropic-Beta") {
				for _, beta := range strings.Split(value, ",") {
					if strings.TrimSpace(beta) == "thinking-display-updates-2026-08-18" {
						return nil
					}
				}
			}
			return ClaudeParameterError{Param: "thinking.display", Message: "thinking.display=updates requires thinking-display-updates-2026-08-18 in the outgoing Anthropic-Beta header"}
		}
	}
	return ClaudeParameterError{Param: "thinking.display", Message: "thinking.display must be summarized, omitted, or updates with its beta header"}
}

var claudeParameterSchemaError = regexp.MustCompile(`^((?:thinking|output_config|messages|tools|tool_choice|system|context_management)(?:\.[A-Za-z0-9_]+)*): (?:Input should be |Extra inputs are not permitted)`)

// ClassifyClaudeParameterError only recognizes field-level schema refusals, not channel failures.
func ClassifyClaudeParameterError(status int, body []byte) error {
	if status != http.StatusBadRequest || ojson.GetBytesNoCopy(body, "error.type").String() != "invalid_request_error" {
		return nil
	}
	match := claudeParameterSchemaError.FindStringSubmatch(ojson.GetBytesNoCopy(body, "error.message").String())
	if len(match) != 2 {
		return nil
	}
	return ClaudeParameterError{Param: match[1], Message: "The upstream rejected an unsupported request parameter: " + match[1]}
}
