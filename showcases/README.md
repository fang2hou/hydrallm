# Showcases

Sample HydraLLM configuration files.

## GLM Coding Plan Showcase

| File | Description |
|------|-------------|
| `glm-coding-plan.toml` | Two-listener setup for OpenAI-compatible and Anthropic-compatible APIs |

### Listener Ports

- OpenAI-compatible listener: `8101`
- Anthropic-compatible listener: `8102`

## Quick Usage

```bash
mkdir -p ~/.config/hydrallm
cp showcases/glm-coding-plan.toml ~/.config/hydrallm/config.toml
export ZAI_API_KEY="your-api-key"
hydrallm
```

## What This Showcase Demonstrates

- Multi-provider fallback (Global + Mainland China routes)
- Multi-model fallback (`glm-5` then `glm-4.7`)
- Multi-listener routing by API format
- Environment-variable-based API key (`$ZAI_API_KEY`)

## Fallback Order

Requests follow the model order defined for each listener.

### OpenAI-compatible listener (`8101`)

1. `glm-5` (Global) - 5 attempts
2. `glm-5` (China) - 2 attempts
3. `glm-4.7` (Global) - 2 attempts
4. `glm-4.7` (China) - 2 attempts

### Anthropic-compatible listener (`8102`)

1. `glm-5` (Global) - 5 attempts
2. `glm-5` (China) - 2 attempts
3. `glm-4.7` (Global) - 2 attempts
4. `glm-4.7` (China) - 2 attempts

If a model exhausts all attempts, HydraLLM falls back to the next model in the same listener.
