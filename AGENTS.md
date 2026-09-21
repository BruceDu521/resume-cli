# Project context

- Before working on this assessment, read `.local/requirements-source.md`, `.local/requirements-checklist.md`, and `.local/session-notes.md` if present. These are local reference files and are intentionally excluded from Git.
- The source assessment is confidential. Do not publish its full text or the local reference files. The eventual implementation and public README should describe the project independently.
- Keep API keys and real candidate documents out of Git. Use synthetic fixtures for committed examples and tests.
- Language is Go. Read docs/development.md, docs/architecture.md and docs/evaluation.md for the current implementation; read .local/session-notes.md for run history.
- Latest user decision (2026-09-21): remove Jev from THIS repository. The other session owns an independent Jev implementation; do not touch its directory. This supersedes earlier instructions to preserve the optional pipeline.
- Public extract passes full PDF text and ResumeSchema directly, no numbered blocks/facts. Skill extraction follows actual described work. The former 64-fact cap was arbitrary and has been removed. Score independently consumes full resume text and JD.
- All five bonus features remain mandatory, alongside Chinese/English reports and broad offline Go tests. Preserve small reusable interfaces and keep unit tests off real networks.
- The user authorized task-specific DeepSeek/Gemini/Kimi Code testing, and the named real PDF under career-2026/output/pdf. Kimi is a Code subscription key: https://api.kimi.com/coding/v1 with model k3. OpenAI is deferred. Never discover credentials elsewhere.
- Latest score simplification: default single sends complete resume text + JD and returns four model-assessed scores, comment and questions (model-assessment-v1). No matches/facts/block IDs/quotes in this route. Validate JSON, integer ranges and nonempty prose; do not claim semantic or coverage guarantees. Preserve original validation reasons when corrective requests fail. No new live calls in the simplification turn; old outputs do not validate the new contract.
