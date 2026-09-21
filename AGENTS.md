# Project context

- Before working on this assessment, read `.local/requirements-source.md`, `.local/requirements-checklist.md`, and `.local/session-notes.md` if present. These are local reference files and are intentionally excluded from Git.
- The source assessment is confidential. Do not publish its full text or the local reference files. The eventual implementation and public README should describe the project independently.
- Keep API keys and real candidate documents out of Git. Use synthetic fixtures for committed examples and tests.
- Language is Go. Read docs/development.md, docs/architecture.md and docs/evaluation.md for the current implementation; read .local/session-notes.md for run history.
- Latest clarification: single-model is the DEFAULT, not the only mode. Preserve Jev behind explicit --pipeline jev (hybrid alias), with its separate optional key. Prior removal was a misunderstanding. Focus current development/testing on the single-model route.
- Public extract passes full PDF text and ResumeSchema directly, no numbered blocks/facts. Skill extraction follows actual described work. The former 64-fact cap was arbitrary and has been removed. Scoring evidence is a separate internal concern.
- All five bonus features remain mandatory, alongside Chinese/English reports and broad offline Go tests. Preserve small reusable interfaces and keep unit tests off real networks.
- The user authorized task-specific DeepSeek/Gemini/Kimi Code testing, and the named real PDF under career-2026/output/pdf. Kimi is a Code subscription key: https://api.kimi.com/coding/v1 with model k3. OpenAI is deferred. Never discover credentials elsewhere.
- Explicit source ranges (block_id/end_block_id) fix the former single-line-only evidence contract. Keep contiguous source validation and do not silently relocate incorrect citations. Historical evaluation docs predate this change.
