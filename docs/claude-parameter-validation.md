# Claude parameter validation

The Claude executor validates `thinking.display` after payload rules and outgoing headers have been finalized, before sending generation, streaming generation, or upstream token-count requests. It rejects unknown display values, display combined with `disabled` or `between_tools`, and `updates` when the final `Anthropic-Beta` header lacks `thinking-display-updates-2026-08-18`. It does not change the payload or add a missing beta. An absent or null display remains accepted.

A valid `updates` request with its final beta header remains supported. This matters for count requests because the header policy can replace caller betas with a baseline or an operator can strip a beta. Validation checks the emitted headers, rather than assuming that an incoming header survived.

These failures return HTTP 400 with `error.type` set to `invalid_request_error`, `error.code` set to `unsupported_request_parameter`, and `error.param` naming the field. They stop credential rotation and retries even when a request-scoped error rule would otherwise continue. Payload rules can still remove or correct an incompatible field before validation.

The same terminal response is used for upstream HTTP 400 `invalid_request_error` responses containing an anchored field path under `thinking`, `output_config`, `messages`, `tools`, `tool_choice`, `system`, or `context_management`, followed by a complete quoted `Input should be` enum diagnostic or exactly `Extra inputs are not permitted`. Additional diagnostic text or lines do not match. That first upstream request cannot be prevented when the incompatibility was not known locally, but it is not replayed. The response names the field without copying the upstream diagnostic. Other errors retain the existing retry policy.

The guard is deliberately limited to confirmed protocol constraints. It does not reject arbitrary new fields or maintain a blanket model capability blacklist. Custom gateways using the Claude executor receive the same display validation.

The display contract was checked against [Anthropic's thinking documentation](https://platform.claude.com/docs/en/build-with-claude/thinking) on 2026-11-11. Tests exercise the final HTTP transport boundary and the auth manager with two credentials and retries enabled. They use controlled transports and do not prove live provider acceptance.
