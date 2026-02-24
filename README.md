# HydraLLM

LLM API proxy with automatic retry and model fallback.

When a request fails, it automatically tries the next configured model until success or all models exhausted.

## Install

```bash
go install github.com/fang2hou/hydrallm@latest
```

## Try It

### Using GLM-5 Coding Plan

**macOS / Linux:**

```bash
mkdir -p ~/.config/hydrallm
curl -o ~/.config/hydrallm/config.toml \
  https://raw.githubusercontent.com/fang2hou/hydrallm/main/showcases/glm-5-coding-plan-openai.toml

export ZAI_API_KEY="your-api-key"
hydrallm
```

**Windows (PowerShell):**

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.config\hydrallm"
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/fang2hou/hydrallm/main/showcases/glm-5-coding-plan-openai.toml" -OutFile "$env:USERPROFILE\.config\hydrallm\config.toml"

$env:ZAI_API_KEY = "your-api-key"
hydrallm
```

Server runs at `http://127.0.0.1:8101`.

### Custom Setup

```bash
hydrallm edit    # Opens config template in $EDITOR
hydrallm         # Start server
```

## Configuration

Config file: `~/.config/hydrallm/config.toml`

### Minimal Example

```toml
[endpoints.openai]
url = "https://api.openai.com/v1"
api_key = "$OPENAI_API_KEY"   # Use $ prefix for env vars

[[models]]
endpoint = "openai"
type = "openai"
model = "gpt-4o"
attempts = 3
```

### Model Fallback

Models are tried in order. Each can have multiple retry attempts:

```toml
[[models]]
endpoint = "openai"
model = "gpt-4o"
attempts = 3

[[models]]
endpoint = "azure"
model = "gpt-4o"
attempts = 2
```

<details>
<summary><b>API Types</b></summary>

**OpenAI** (default)
```toml
[endpoints.openai]
url = "https://api.openai.com/v1"
api_key = "$OPENAI_API_KEY"
```

**Anthropic**
```toml
[endpoints.anthropic]
url = "https://api.anthropic.com/v1"
api_key = "$ANTHROPIC_API_KEY"

[[models]]
endpoint = "anthropic"
type = "anthropic"
model = "claude-3-5-sonnet-20241022"
```

**AWS Bedrock**
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
```

</details>

<details>
<summary><b>Full Options</b></summary>

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

</details>

## Commands

```bash
hydrallm           # Start server
hydrallm edit      # Edit config
hydrallm version   # Show version
```

## License

MIT
