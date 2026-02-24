# HydraLLM

HydraLLM is a high-performance LLM API proxy with automatic retry and model fallback.

When a request fails, HydraLLM retries the same model and falls back to the next configured model until a request succeeds or all attempts are exhausted.

## ✨ Why HydraLLM

- Improve reliability for coding workflows and agent workloads.
- Reduce failed requests caused by transient upstream errors.
- Keep one local API entrypoint while switching providers/models behind the scenes.
- Support OpenAI-compatible, Anthropic, and AWS Bedrock endpoints.

## 📦 Install

### Homebrew (macOS / Linux)

```bash
brew install fang2hou/tap/hydrallm
```

### Binary Download (All platforms)

Download the latest release from [GitHub Releases](https://github.com/fang2hou/hydrallm/releases).

### Install via Go (All platforms)

```bash
go install github.com/fang2hou/hydrallm@latest
```

## 🚀 Quick Start (GLM Coding Plan)

Choose one quick start below.

<details>
<summary><b>GLM OpenAI Quick Start</b></summary>

### 1) Prepare config

**macOS / Linux:**

```bash
mkdir -p ~/.config/hydrallm
curl -o ~/.config/hydrallm/config.toml \
  https://raw.githubusercontent.com/fang2hou/hydrallm/main/showcases/glm-5-coding-plan-openai.toml
```

**Windows (PowerShell):**

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.config\hydrallm"
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/fang2hou/hydrallm/main/showcases/glm-5-coding-plan-openai.toml" -OutFile "$env:USERPROFILE\.config\hydrallm\config.toml"
```

### 2) Set API key

**macOS / Linux:**

```bash
export ZAI_API_KEY="your-api-key"
```

**Windows (PowerShell):**

```powershell
$env:ZAI_API_KEY = "your-api-key"
```

### 3) Start proxy

```bash
hydrallm
```

This showcase listens on `http://127.0.0.1:8101`.

### 4) Verify with a request

```bash
curl http://127.0.0.1:8101/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "placeholder",
    "messages": [{"role": "user", "content": "Say hello"}]
  }'
```

</details>

<details>
<summary><b>GLM Anthropic Quick Start</b></summary>

### 1) Prepare config

**macOS / Linux:**

```bash
mkdir -p ~/.config/hydrallm
curl -o ~/.config/hydrallm/config.toml \
  https://raw.githubusercontent.com/fang2hou/hydrallm/main/showcases/glm-5-coding-plan-anthropic.toml
```

**Windows (PowerShell):**

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.config\hydrallm"
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/fang2hou/hydrallm/main/showcases/glm-5-coding-plan-anthropic.toml" -OutFile "$env:USERPROFILE\.config\hydrallm\config.toml"
```

### 2) Set API key

**macOS / Linux:**

```bash
export ZAI_API_KEY="your-api-key"
```

**Windows (PowerShell):**

```powershell
$env:ZAI_API_KEY = "your-api-key"
```

### 3) Start proxy

```bash
hydrallm
```

This showcase listens on `http://127.0.0.1:8102`.

### 4) Verify with a request

```bash
curl http://127.0.0.1:8102/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "placeholder",
    "max_tokens": 64,
    "messages": [{"role": "user", "content": "Say hello"}]
  }'
```

</details>

HydraLLM overrides the request `model` based on your configured model chain.

## 🔁 Service (Auto Start on Boot)

Use Homebrew service for auto-start.

### Start

```bash
brew services start hydrallm
```

### Status

```bash
brew services info hydrallm
```

### Restart / Stop

```bash
brew services restart hydrallm
brew services stop hydrallm
```

Notes:

- On macOS, `brew services start hydrallm` runs via `launchd` (auto-start after login).
- On Linux, `brew services` uses `systemd`.
- For `brew services`, configure `api_key` explicitly in the config file; do not rely on shell environment variables.

## ⚙️ Configuration

For full configuration details, see [CONFIGURATION.md](CONFIGURATION.md).

## 🛠️ CLI Commands

```bash
hydrallm                 # Start server
hydrallm serve           # Start server
hydrallm edit            # Open config in $EDITOR
hydrallm version         # Print version info
hydrallm --help
```

Global flags:

```bash
hydrallm --config /path/to/config.toml
hydrallm --host 127.0.0.1 --port 8080 --log-level info
```

## 🧯 Troubleshooting

Quick diagnostics:

```bash
hydrallm --config /path/to/config.toml --log-level debug
brew services list | grep hydrallm
```

<details>
<summary><b>config validation failed: at least one model must be configured</b></summary>

Add at least one `[[models]]` entry in your config file.

</details>

<details>
<summary><b>endpoint "..." not found</b></summary>

Ensure each model `endpoint` matches a key under `[endpoints.&lt;name&gt;]`.

</details>

<details>
<summary><b>Service starts but auth fails</b></summary>

The service manager (`launchd`/`systemd`) may not have your shell environment variables. Use explicit `api_key` values in your config for service mode.

</details>

<details>
<summary><b>Requests return upstream 4xx/5xx</b></summary>

Temporarily set `log.include_error_body = true` to inspect upstream error responses.

</details>

## 📄 License

MIT
