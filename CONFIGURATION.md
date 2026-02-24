# Configuration Guide

This document contains detailed configuration for HydraLLM.

## Config File Location

- macOS / Linux: `~/.config/hydrallm/config.toml`
- Windows: `%USERPROFILE%\.config\hydrallm\config.toml`

Create or edit the config file with:

```bash
hydrallm edit
```

## Minimal Example

```toml
[endpoints.openai]
url = "https://api.openai.com/v1"
api_key = "$OPENAI_API_KEY"

[[models]]
endpoint = "openai"
type = "openai"
model = "gpt-4o"
attempts = 3
```

## Model Type Rule (Important)

All entries in `[[models]]` must use the same `type`.

- ✅ Allowed: all `openai`, or all `anthropic`, or all `bedrock`
- ❌ Not allowed: mixing types in one config (for example `openai` + `anthropic`)

HydraLLM validates this at startup and exits if mixed types are configured.

## Retry and Fallback Behavior

Models are tried in order.

- For each model, HydraLLM retries up to `attempts`.
- Then it falls back to the next model.
- This process repeats for up to `retry.max_cycles`.

### Example: OpenAI-compatible fallback chain

```toml
[endpoints.primary]
url = "https://api.openai.com/v1"
api_key = "$OPENAI_API_KEY"

[endpoints.backup]
url = "https://api.openai-proxy.example.com/v1"
api_key = "$BACKUP_OPENAI_API_KEY"

[[models]]
endpoint = "primary"
type = "openai"
model = "gpt-4o"
attempts = 2

[[models]]
endpoint = "backup"
type = "openai"
model = "gpt-4o"
attempts = 2
```

In this example, HydraLLM tries `primary` first, then falls back to `backup`.

### Example: Anthropic fallback chain

```toml
[endpoints.anthropic-primary]
url = "https://api.anthropic.com/v1"
api_key = "$ANTHROPIC_API_KEY"

[endpoints.anthropic-backup]
url = "https://anthropic-proxy.example.com/v1"
api_key = "$ANTHROPIC_BACKUP_API_KEY"

[[models]]
endpoint = "anthropic-primary"
type = "anthropic"
model = "claude-3-5-sonnet-20241022"
attempts = 2

[[models]]
endpoint = "anthropic-backup"
type = "anthropic"
model = "claude-3-5-sonnet-20241022"
attempts = 2
```

## API Key Resolution

HydraLLM resolves authentication as follows:

1. Use endpoint `api_key` when configured.
2. Otherwise, keep the original incoming auth header.

Use `api_key = "-"` to explicitly remove auth for a specific endpoint.

## Provider Examples

### OpenAI-compatible

```toml
[endpoints.openai]
url = "https://api.openai.com/v1"
api_key = "$OPENAI_API_KEY"

[[models]]
endpoint = "openai"
type = "openai"
model = "gpt-4o"
attempts = 3
```

### Anthropic

```toml
[endpoints.anthropic]
url = "https://api.anthropic.com/v1"
api_key = "$ANTHROPIC_API_KEY"

[[models]]
endpoint = "anthropic"
type = "anthropic"
model = "claude-3-5-sonnet-20241022"
attempts = 2
```

### AWS Bedrock

```toml
[endpoints.bedrock]
url = "https://bedrock-runtime.us-east-1.amazonaws.com"
aws_region = "us-east-1"
aws_access_key_id = "$AWS_ACCESS_KEY_ID"
aws_secret_access_key = "$AWS_SECRET_ACCESS_KEY"
aws_session_token = "$AWS_SESSION_TOKEN"

[[models]]
endpoint = "bedrock"
type = "bedrock"
model = "anthropic.claude-3-sonnet-20240229-v1:0"
attempts = 2
```

## Full Option Reference

```toml
[server]
host = "127.0.0.1"
port = 8080
read_timeout = "60s"
write_timeout = "10m"

[log]
level = "info"              # debug, info, warn, error
include_error_body = false

[retry]
max_cycles = 10
default_timeout = "30s"
default_interval = "100ms"
exponential_backoff = false

[[models]]
endpoint = "openai"
type = "openai"
model = "gpt-4o"
attempts = 3
timeout = "30s"             # optional override
interval = "200ms"          # optional override
```
