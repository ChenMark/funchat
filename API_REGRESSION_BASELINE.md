# FunChat API Regression Baseline

This baseline reflects the hardened backend on branch `codex/production-hardening` as of commit `dc2251c90db1097af40c60d6226e199577689774`.

## Goal

Use this document to distinguish:

- supported flows that must keep working
- intentionally gated or unavailable flows that must fail clearly
- environment-dependent behavior that changes between `development` and `production`

## How To Run

Preferred:

```bash
docker compose up -d
./test_integration.sh
```

If the backend runs on a different host or port:

```bash
BASE_URL=http://localhost:8080 ./test_integration.sh
```

If you want to validate production-style third-party login behavior:

```bash
THIRD_PARTY_LOGIN_EXPECTED_HTTP=501 \
THIRD_PARTY_LOGIN_EXPECTED_CODE=50100 \
./test_integration.sh
```

## Core Flows That Must Pass

- `GET /health`
- SMS auth flow: `send-code` -> `verify-code` -> `register` -> `login` -> `refresh`
- friend flow: search -> request -> accept -> list -> delete
- direct message flow: send -> history -> conversations
- burn-after-reading flow: send-burn -> pending -> burn-read -> burn-status -> burn-destroy
- supported condition storage flow: `conditions/set`, `conditions/:user_id`
- quiz flow: `set-quiz`, `quiz/:id`, `verify-quiz`

## Flows That Must Be Explicitly Rejected

These are no longer allowed to pretend success:

- media upload endpoints return HTTP `501`, code `50101`
- conditional message endpoints return HTTP `501`, code `50102`
- location and steps verification endpoints return HTTP `501`, code `50102`
- sending a normal message to a non-friend returns HTTP `403`, code `40300`

## Environment-Specific Expectation

Third-party auth is intentionally split by environment:

- `development`: WeChat and Apple login return HTTP `200`, code `0`, with `need_bind=true`
- `production`: WeChat and Apple login return HTTP `501`, code `50100`

## Why The Old Script Was No Longer Reliable

The previous integration script had three problems:

- it expected several gated prototype endpoints to succeed
- it hardcoded friend request IDs instead of reading the API response
- it only checked business `code`, which could hide HTTP-level regressions

The rewritten `test_integration.sh` fixes those issues and matches the current backend contract.
