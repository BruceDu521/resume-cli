# Project context

- Before working on this assessment, read `.local/requirements-source.md`, `.local/requirements-checklist.md`, and `.local/session-notes.md` if present. These are local reference files and are intentionally excluded from Git.
- The source assessment is confidential. Do not publish its full text or the local reference files. The eventual implementation and public README should describe the project independently.
- Keep API keys and real candidate documents out of Git. Use synthetic fixtures for committed examples and tests.
- Language is Go, explicitly chosen by the user. Offline implementation and tests are complete; see `docs/architecture.md` for the implemented design. Live model verification and evaluation still await task-specific API keys.
- Resume development from `docs/development.md`, `docs/architecture.md`, and `docs/evaluation.md`. Keep README status consistent with actual implemented behavior.
- Compare Gemini 3.8 Flash and DeepSeek V4.1 Flash on content quality AND real latency; prefer DeepSeek when comparable. Gemini is not a settled default. OpenAI and Kimi are independent full-pipeline comparison baselines, not merely fallbacks.
- The user prioritizes maintainable, reusable Go code and cost/latency-aware model composition. All five original bonus items are mandatory; do not drop them to optional scope.
- Jev handles focused semantic judgments within a local-PDF/structured-data/report pipeline. Other AI models may assist. Do not revisit whether one model must do everything; the user explicitly allows multiple models.
- The user requested evaluating Jev. Its installed skill is `/Users/brucedu/.agents/skills/typesafe-ai/SKILL.md`; consult it and current official documentation before making API or capability assumptions.

- The user confirmed Chinese reports by default and English switching, and authorized development with broad offline Go unit tests. No real API calls until task-specific keys are supplied.
