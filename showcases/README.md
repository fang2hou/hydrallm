# Showcases

Sample configurations for HydraLLM.

## GLM-5 Coding Plan

| File | Format | Port |
|------|--------|------|
| `glm-5-coding-plan-openai.toml` | OpenAI | 8101 |
| `glm-5-coding-plan-anthropic.toml` | Anthropic | 8102 |

**Features:**
- Dual endpoint (Global + China)
- Model fallback (`glm-5` → `glm-4.7`)
- Env var for API key (`$ZAI_API_KEY`)

**Model Priority:**
1. glm-5 (Global) - 5 attempts
2. glm-5 (China) - 2 attempts
3. glm-4.7 (Global) - 2 attempts
4. glm-4.7 (China) - 2 attempts
