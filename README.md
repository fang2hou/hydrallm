# HydraLLM

HydraLLM is a high-performance LLM API proxy with automatic retry and model fallback across OpenAI-compatible, Anthropic, and AWS Bedrock providers.

When a request fails, HydraLLM retries the current model, then falls back to the next configured model until success or exhaustion.

> [!TIP]
> For complete configuration fields and examples, see [CONFIGURATION.md](CONFIGURATION.md).

## ✨ Why HydraLLM

- Automatic retry + fallback for coding and agent workloads.
- Multi-provider support: OpenAI-compatible, Anthropic, and AWS Bedrock.
- Single local endpoint with stable client integration while model chains evolve.

## 📦 Install

### Homebrew (macOS / Linux)

```bash
brew install fang2hou/tap/hydrallm
```

### Install via Go (All platforms)

```bash
go install github.com/fang2hou/hydrallm@latest
```

### Binary Download (All platforms)

Download from [GitHub Releases](https://github.com/fang2hou/hydrallm/releases).

## 🚀 Quick Start (GLM Coding Plan)

The official GLM showcase exposes two listeners in one config:

- OpenAI-compatible API on `http://127.0.0.1:8101`
- Anthropic-compatible API on `http://127.0.0.1:8102`

### 1) Prepare config

**macOS / Linux:**

```bash
mkdir -p ~/.config/hydrallm
curl -o ~/.config/hydrallm/config.toml \
  https://raw.githubusercontent.com/fang2hou/hydrallm/main/showcases/glm-coding-plan.toml
```

**Windows (PowerShell):**

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.config\hydrallm"
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/fang2hou/hydrallm/main/showcases/glm-coding-plan.toml" -OutFile "$env:USERPROFILE\.config\hydrallm\config.toml"
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

### 4) Verify listeners

<details>
<summary><b>OpenAI-compatible listener (8101)</b></summary>

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
<summary><b>Anthropic-compatible listener (8102)</b></summary>

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

> [!NOTE]
> HydraLLM overrides request `model` with the configured model chain for each listener.

## 🔁 Service (Auto Start on Boot)

Use Homebrew services for auto-start.

> [!NOTE]
> For `brew services`, configure `api_key` explicitly in the config file; do not rely on shell environment variables.

```bash
brew services start hydrallm
brew services info hydrallm
brew services restart hydrallm
brew services stop hydrallm
```

- macOS: `launchd` (starts automatically after login)
- Linux: `systemd`

## 🛠️ CLI Commands

| Command | Description |
|---|---|
| `hydrallm` | Start server |
| `hydrallm serve` | Start proxy |
| `hydrallm edit` | Open config in `$EDITOR` |
| `hydrallm version` | Print version info |
| `hydrallm --help` | Show help |

Global flags: `--config /path/to/config.toml`, `--log-level info`

## 🧯 Troubleshooting

Quick diagnostics:

```bash
hydrallm --config /path/to/config.toml --log-level debug
brew services list | grep hydrallm
```

<details>
<summary><b>config validation failed: at least one model must be configured</b></summary>

Add at least one model under `[models.<id>]`.

</details>

<details>
<summary><b>model "...": provider "..." not found</b></summary>

Ensure each model `provider` matches a key under `[providers.<name>]`.

</details>

<details>
<summary><b>listener "...": mixed model types are not allowed</b></summary>

Each listener must contain models of a single API type (`openai`, `anthropic`, or `bedrock`).
Split mixed types across multiple listeners.

</details>

<details>
<summary><b>Requests return upstream 4xx/5xx</b></summary>

Temporarily set `log.include_error_body = true` to inspect upstream error details.

</details>

## 📄 License

MIT
