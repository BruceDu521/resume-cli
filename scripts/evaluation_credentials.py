"""Explicit, non-executable credential profiles for offline/real evaluations."""
import os


def load_credentials(providers, env_dir=None, environ=None):
    """Return only the keys needed for this run; never infer vendor from a key."""
    providers = set(providers) - {""}
    environ = os.environ if environ is None else environ
    if not providers:
        return {}
    if env_dir is None:
        if len(providers) != 1:
            raise ValueError("multiple providers require --env-dir with separate provider profiles")
        key = environ.get("RESUME_AI_API_KEY", "").strip()
        if not key:
            raise ValueError("missing RESUME_AI_API_KEY")
        return {next(iter(providers)): key}
    result = {}
    for provider in sorted(providers):
        if provider not in {"gemini", "deepseek", "kimi", "openai"}:
            raise ValueError("unsupported provider profile")
        try:
            lines = (env_dir / (provider + ".env")).read_text().splitlines()
        except OSError:
            raise ValueError("cannot read credential profile for " + provider) from None
        values = []
        for line in lines:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            name, sep, value = line.partition("=")
            if not sep or name.strip() != "RESUME_AI_API_KEY":
                raise ValueError("profile must contain only RESUME_AI_API_KEY for " + provider)
            value = value.strip()
            if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
                value = value[1:-1]
            values.append(value)
        if len(values) != 1 or not values[0]:
            raise ValueError("profile requires one nonempty RESUME_AI_API_KEY for " + provider)
        result[provider] = values[0]
    return result
