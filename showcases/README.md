# Showcases

Sample configurations for HydraLLM.

## GLM-5 Coding Plan

| File | Format | Port |
|------|--------|------|
| `glm-5-coding-plan-openai.toml` | OpenAI | 8101 |
| `glm-5-coding-plan-anthropic.toml` | Anthropic | 8102 |

## Quick Usage

Pick one showcase and copy it to your config path:

```bash
mkdir -p ~/.config/hydrallm
cp showcases/glm-5-coding-plan-openai.toml ~/.config/hydrallm/config.toml
export ZAI_API_KEY="your-api-key"
hydrallm
```

For Anthropic format, replace the file with `showcases/glm-5-coding-plan-anthropic.toml`.

## Important Rule

All `[[models]]` in one config must use the same `type`.

- `glm-5-coding-plan-openai.toml` uses `type = "openai"` only.
- `glm-5-coding-plan-anthropic.toml` uses `type = "anthropic"` only.

Do not mix OpenAI and Anthropic model types in a single config file.

**Features:**
- Dual endpoint (Global + China)
- Model fallback (`glm-5` → `glm-4.7`)
- Env var for API key (`$ZAI_API_KEY`)

## Fallback Order

Each request follows this order:

1. `glm-5` (Global) - 5 attempts
2. `glm-5` (China) - 2 attempts
3. `glm-4.7` (Global) - 2 attempts
4. `glm-4.7` (China) - 2 attempts

If a model fails after all attempts, HydraLLM falls back to the next model in the list.
