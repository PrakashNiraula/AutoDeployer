# Operator walkthrough (CLI-only, Phase 1)

Install root layout: `app/` (binary), `project/` (jobs), `data/` (local machine state).

```bash
# From the install root
./app/autodeployer version --json
./app/autodeployer doctor --json
./app/autodeployer schema validate all --json
./app/autodeployer jobs list --json
```

## Env

```bash
./app/autodeployer env list --job db-backup --env dev --json
./app/autodeployer env set --job db-backup --env dev --key DB_PASSWORD --value '***'
# values never appear in history.db env_audit
./app/autodeployer env get --job db-backup --env dev --key DB_PASSWORD          # masked
./app/autodeployer env get --job db-backup --env dev --key DB_PASSWORD --reveal # cleartext
```

## Plan → run → history

```bash
./app/autodeployer plan db-backup --env dev --json > /tmp/plan.json
PLAN_ID=$(python3 -c 'import json;print(json.load(open("/tmp/plan.json"))["plan_id"])')

./app/autodeployer run db-backup --env dev --trigger manual --plan-id "$PLAN_ID" --yes \
  --json-meta data/output/db-backup/last-meta.json \
  --events data/output/db-backup/last-events.ndjson

./app/autodeployer history list --job db-backup --json
./app/autodeployer history show <run-id> --json
./app/autodeployer logs tail <run-id>
```

## Cancel a running job

```bash
./app/autodeployer history list --json   # find status=running
./app/autodeployer cancel <run-id> --json
```

## Doctor

```bash
./app/autodeployer doctor --json
./app/autodeployer doctor --fix --json
```

Back up the absolute `data/` path printed by doctor (history, env, schedules).
