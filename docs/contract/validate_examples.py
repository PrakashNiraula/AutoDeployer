#!/usr/bin/env python3
"""Validate contract examples against JSON Schemas (draft 2020-12)."""

from __future__ import annotations

import json
import sys
from pathlib import Path

try:
    import yaml
    from jsonschema import Draft202012Validator
except ImportError as e:
    print("Missing dependency:", e, file=sys.stderr)
    print("Install with: python3 -m venv .venv && .venv/bin/pip install -r docs/contract/requirements.txt", file=sys.stderr)
    sys.exit(2)

ROOT = Path(__file__).resolve().parent
EXAMPLES = ROOT / "examples"


def load_schema(name: str) -> dict:
    with (ROOT / name).open(encoding="utf-8") as f:
        return json.load(f)


def validate_json(schema_name: str, data: object, label: str) -> list[str]:
    schema = load_schema(schema_name)
    validator = Draft202012Validator(schema)
    return [f"{label}: {e.message}" for e in sorted(validator.iter_errors(data), key=lambda e: list(e.path))]


def main() -> int:
    errors: list[str] = []

    with (EXAMPLES / "plan_result.json").open(encoding="utf-8") as f:
        errors += validate_json("plan_result.schema.json", json.load(f), "plan_result.json")

    with (EXAMPLES / "run_result.json").open(encoding="utf-8") as f:
        errors += validate_json("run_result.schema.json", json.load(f), "run_result.json")

    with (EXAMPLES / "job.yaml").open(encoding="utf-8") as f:
        errors += validate_json("job.schema.json", yaml.safe_load(f), "job.yaml")

    with (EXAMPLES / "machine.yaml").open(encoding="utf-8") as f:
        errors += validate_json("machine.schema.json", yaml.safe_load(f), "machine.yaml")

    event_schema = load_schema("ndjson_event.schema.json")
    event_validator = Draft202012Validator(event_schema)
    with (EXAMPLES / "events.ndjson").open(encoding="utf-8") as f:
        for i, line in enumerate(f, start=1):
            line = line.strip()
            if not line:
                continue
            obj = json.loads(line)
            for e in sorted(event_validator.iter_errors(obj), key=lambda e: list(e.path)):
                errors.append(f"events.ndjson:{i}: {e.message}")

    if errors:
        print("FAIL")
        for err in errors:
            print(" -", err)
        return 1

    print("OK: all contract examples validate")
    return 0


if __name__ == "__main__":
    sys.exit(main())
