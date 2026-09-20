#!/usr/bin/env python3
"""Reproducible evaluation runner. Dry-run by default; never reads .env files."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import random
import statistics
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
PATHS = {
    "gemini": ("gemini", "hybrid", "gemini-3.8-flash"),
    "deepseek": ("deepseek", "hybrid", "deepseek-flash"),
    "openai": ("openai", "baseline", "gpt-6-astra"),
    "kimi": ("kimi", "baseline", "kimi-k3"),
    "mock": ("", "hybrid", ""),
}


def write_json(path, value):
    with path.open("x", encoding="utf-8") as f:
        json.dump(value, f, ensure_ascii=False, indent=2)
        f.write("\n")


def summarize(rows):
    summary = {}
    for route in sorted({r["route"] for r in rows}):
        group = [r for r in rows if r["route"] == route]
        times = [r["wall_ms"] for r in group]
        calls = [c for r in group for c in r["stats"].get("calls", [])]
        complete = all(r["success"] for r in group) and all(c.get("cost_complete", False) for c in calls)
        if route != "mock" and not calls:
            complete = False
        summary[route] = {
            "runs": len(group), "successes": sum(r["success"] for r in group),
            "wall_ms_including_failures": {"median": statistics.median(times), "min": min(times), "max": max(times)},
            "observed_cost_usd": sum(c.get("estimated_cost_usd", 0) for c in calls),
            "cost_complete": complete,
            "request_attempts": sum(c.get("attempts", 0) for c in calls),
            "json_repairs": sum(c.get("json_repaired", False) for c in calls),
            "quality": "pending human review; no automatic winner",
        }
    return summary


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--execute", action="store_true", help="actually run the CLI; real routes incur API calls")
    p.add_argument("--routes", nargs="+", choices=PATHS, default=["gemini", "deepseek", "openai", "kimi"])
    p.add_argument("--repeats", type=int, default=3)
    p.add_argument("--seed", type=int, default=20260920)
    p.add_argument("--limit", type=int, help="use only the first N cases for a smoke test")
    p.add_argument("--binary", type=Path, default=ROOT / "bin/resume-cli")
    p.add_argument("--suite", type=Path, default=ROOT / "testdata/evaluation/cases.json", help="case manifest; use a separate suite for held-out validation")
    p.add_argument("--capture-structures", action="store_true", help="save validated structures in a fresh private cache per trial; never reuse across trials")
    p.add_argument("--out", type=Path, help="new private output directory; required with --execute")
    args = p.parse_args()
    if not 1 <= args.repeats <= 10 or (args.limit is not None and args.limit < 1):
        p.error("repeats must be 1..10; limit must be positive")
    cases = json.loads(args.suite.read_text(encoding="utf-8"))
    if "mock" in args.routes:
        if args.routes != ["mock"]:
            p.error("mock smoke results must be kept separate from real model results")
        cases = [c for c in cases if c.get("mock")]
    if args.limit:
        cases = cases[:args.limit]
    jobs = [(c, r, n) for c in cases for r in dict.fromkeys(args.routes) for n in range(1, args.repeats + 1)]
    random.Random(args.seed).shuffle(jobs)
    sources = sorted([*ROOT.glob("internal/**/*.go"), *ROOT.glob("cmd/**/*.go"), ROOT / "go.mod", ROOT / "go.sum"])
    digest = hashlib.sha256()
    for source in sources:
        digest.update(str(source.relative_to(ROOT)).encode() + b"\0" + source.read_bytes())
    plan = {
        "seed": args.seed, "cache": "fresh per trial" if args.capture_structures else "disabled", "code_sha256": digest.hexdigest(),
        "runner_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        "manifest_sha256": hashlib.sha256(args.suite.read_bytes()).hexdigest(),
        "models": {r: PATHS[r] for r in args.routes}, "jev": "jev-1.13.0",
        "inputs": {c["id"]: {"pdf_sha256": hashlib.sha256((ROOT / c["pdf"]).read_bytes()).hexdigest(),
                              "jd_sha256": hashlib.sha256((ROOT / c["jd"]).read_bytes() if "jd" in c else c["jd_text"].encode()).hexdigest()} for c in cases},
        "jobs": [{"case": c["id"], "route": r, "repeat": n} for c, r, n in jobs],
    }
    if not args.execute:
        print(json.dumps(plan, indent=2))
        print("Dry run only. No CLI execution or API requests.")
        return
    if args.out is None:
        p.error("--out is required with --execute")
    binary = args.binary.resolve()
    if not binary.is_file():
        p.error("build bin/resume-cli first")
    os.umask(0o077)
    args.out.mkdir(parents=True, exist_ok=False)
    write_json(args.out / "plan.json", plan)
    env = os.environ.copy()
    # Pin the benchmark routes. Do not silently inherit custom proxy/model settings.
    for key in ("RESUME_AI_PROVIDER", "RESUME_AI_MODEL", "RESUME_AI_BASE_URL", "TYPESAFE_BASE_URL", "RESUME_JEV_MODEL"):
        env.pop(key, None)
    env["RESUME_JEV_MODEL"] = "jev-1.13.0"
    rows = []
    for case, route, repeat in jobs:
        dest = args.out / f"{case['id']}-{route}-{repeat}"
        dest.mkdir()
        jd = ROOT / case["jd"] if "jd" in case else dest / "jd.txt"
        if "jd" not in case:
            jd.write_text(case["jd_text"], encoding="utf-8")
        cmd = [str(binary), "score", str(ROOT / case["pdf"]), "--jd", str(jd), "--lang", case["lang"],
               "--timeout", "180s", "--output", str(dest / "result.json"), "--stats", str(dest / "stats.json")]
        provider, pipeline, model = PATHS[route]
        cmd += ["--mock"] if route == "mock" else ["--provider", provider, "--pipeline", pipeline, "--model", model]
        if args.capture_structures:
            cmd += ["--cache-dir", str(dest / "structures")]
        start = time.monotonic()
        with (dest / "stderr.log").open("wb") as log:
            try:
                result = subprocess.run(cmd, stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=log, env=env, timeout=195)
                code = result.returncode
            except subprocess.TimeoutExpired:
                code = -1
        stats = json.loads((dest / "stats.json").read_text()) if (dest / "stats.json").exists() else {}
        row = {"case": case["id"], "route": route, "repeat": repeat, "success": code == 0,
               "exit_code": code, "wall_ms": round((time.monotonic() - start) * 1000), "stats": stats}
        rows.append(row)
        write_json(dest / "review.json", {"rubric": case["review"], "fact_errors": None, "missing_requirements": None,
                                          "unsupported_judgments": None, "report_quality_0_to_2": None, "notes": ""})
        # Append progress so interruption does not lose completed trials.
        with (args.out / "runs.jsonl").open("a", encoding="utf-8") as f:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")
        print(f"{case['id']} {route} #{repeat}: {'ok' if code == 0 else 'FAILED'} ({row['wall_ms']} ms)", flush=True)
    write_json(args.out / "summary.json", summarize(rows))
    print(f"Review individual outputs and rubrics in {args.out}; latency alone does not select a model.")


if __name__ == "__main__":
    main()
