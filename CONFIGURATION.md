# Configuration Guide

This guide explains HydraLLM's configuration schema:

- `providers` define upstream API base URLs and credentials.
- `models` define retry settings and model overrides.
- `listeners` define local ports and the model chain used by each port.

## Config File Location

- macOS / Linux: `~/.config/hydrallm/config.toml`
- Windows: `%USERPROFILE%\.config\hydrallm\config.toml`

Create or edit the config file with:

```bash
hydrallm edit
```

## Minimal Working Example

```toml
[providers.openai]
url = "https://api.openai.com/v1"
api_key = "$OPENAI_API_KEY"

[models.gpt_5_3_codex]
provider = "openai"
model = "gpt-5.3-codex"
type = "openai"
attempts = 3

[[listeners]]
name = "openai-main"
host = "127.0.0.1"
port = 8080
models = ["gpt_5_3_codex"]
```

## Validation Rules

### Provider URL Requirements

Each provider `url` must:

- Include a scheme and host
- Use `http` or `https`

Examples:

- ✅ `https://api.openai.com/v1`
- ✅ `http://localhost:8000`
- ❌ `api.openai.com/v1` (missing scheme)
- ❌ `ftp://api.example.com` (unsupported scheme)

### Listener Model Type Rule

Within one listener, all referenced models must share the same `type`.

- ✅ Allowed: all `openai`, all `anthropic`, or all `bedrock` in one listener
- ❌ Not allowed: mixed types in one listener

Different listeners may use different types.

### Listener Uniqueness and Port Rules

Each listener must satisfy all of the following:

- Unique listener `name`
- Unique `host:port` binding
- Port in range `1..65535`

## Retry and Fallback Behavior

For each request, HydraLLM processes the selected listener's model list in order:

1. Try the current model up to `attempts`
2. If needed, move to the next model in that listener
3. Repeat this cycle up to `retry.max_cycles`

Retryable responses are `429` and `5xx`.

## API Key Resolution

HydraLLM resolves authentication in this order:

1. The model's provider `api_key`
2. Original incoming request header

Use `api_key = "-"` to explicitly remove auth for that provider.

## Common Provider Examples

### OpenAI-compatible

```toml
[providers.openai]
url = "https://api.openai.com/v1"
api_key = "$OPENAI_API_KEY"

[models.gpt_5_3_codex]
provider = "openai"
model = "gpt-5.3-codex"
type = "openai"
attempts = 3

[models.gpt_5_2_codex]
provider = "openai"
model = "gpt-5.2-codex"
type = "openai"
attempts = 2

[[listeners]]
name = "openai-main"
port = 8080
models = ["gpt_5_3_codex", "gpt_5_2_codex"]
```

### Anthropic

```toml
[providers.anthropic]
url = "https://api.anthropic.com/v1"
api_key = "$ANTHROPIC_API_KEY"

[models.claude]
provider = "anthropic"
model = "claude-opus-4-6"
type = "anthropic"
attempts = 2

[[listeners]]
name = "anthropic-main"
port = 8081
models = ["claude"]
```

### AWS Bedrock

```toml
[providers.bedrock]
url = "https://bedrock-runtime.us-east-1.amazonaws.com"
aws_region = "us-east-1"
aws_access_key_id = "$AWS_ACCESS_KEY_ID"
aws_secret_access_key = "$AWS_SECRET_ACCESS_KEY"
aws_session_token = "$AWS_SESSION_TOKEN"

[models.claude-bedrock]
provider = "bedrock"
model = "anthropic.claude-opus-4-6-v1:0"
type = "bedrock"
attempts = 2

[[listeners]]
name = "bedrock-main"
port = 8082
models = ["claude-bedrock"]
```

## Full Option Reference

```toml
[log]
level = "info"              # debug, info, warn, error
include_error_body = false

[retry]
max_cycles = 10
default_timeout = "30s"
default_interval = "100ms"
exponential_backoff = false

[providers.<name>]
url = "https://api.example.com/v1"
api_key = "$API_KEY"          # optional, use "-" to remove auth
strip_version_prefix = false  # optional
interval = "100ms"            # optional, provider-level retry interval

# bedrock-specific optional fields
aws_region = "us-east-1"
aws_access_key_id = "$AWS_ACCESS_KEY_ID"
aws_secret_access_key = "$AWS_SECRET_ACCESS_KEY"
aws_session_token = "$AWS_SESSION_TOKEN"

[models.<id>]
provider = "<provider-name>"
model = "<upstream-model-name>"
type = "openai"             # openai | anthropic | bedrock
attempts = 3
timeout = "30s"             # optional, falls back to retry.default_timeout
interval = "200ms"          # optional, overrides provider/retry interval

[[listeners]]
name = "main"
host = "127.0.0.1"          # optional, default 127.0.0.1
port = 8080
read_timeout = "60s"        # optional, default 60s
write_timeout = "10m"       # optional, default 10m
models = ["model-id-1", "model-id-2"]
```

## Operational Notes

- HydraLLM rewrites the outgoing `model` field based on the selected model configuration.
- If you run HydraLLM as a service (`launchd` / `systemd`), prefer explicit `api_key` values over shell-only environment variables.
