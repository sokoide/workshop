# AGENTS.md

## What This Repo Is

Documentation-only workshop collection. No CI, no build pipeline, no deploy. The "code" is Go example files referenced by markdown guides. Each workshop is self-contained under `infra/` or `software/`.

## Markdown Formatting (the only real build step)

```bash
make format
```

Runs three tools in sequence: `prettier --write --parser markdown` → `markdownlint --fix` → `textlint --fix`. Expects `node_modules/` to exist (`pnpm install` or `npm install` first).

- `CLAUDE.md` is explicitly excluded from formatting — do not reformat it.
- `conductor/`, `node_modules/`, `.gomodcache/` are ignored.
- markdownlint config (`.markdownlint.jsonc`) disables line-length, duplicate headings, inline HTML, and emphasis-as-heading rules.
- textlint uses `preset-ja-spacing` for Japanese full/half-width spacing rules.

## Bilingual Files

Every workshop has `_ja.md` and `_en.md` pairs. They must stay in sync structurally. If you edit one, update the other. The root `README.md` (English) and `README_ja.md` (Japanese) are the indexes.

## Go Projects

21 independent `go.mod` files under `*/assets/` directories. Each is a standalone module — there is no workspace-level `go.mod`. To build/test a specific workshop:

```bash
cd infra/assets/redis_leaderboard && go run main.go
cd software/assets/bbs && go test ./...
```

Go version in `go.mod` files may lag behind the repo-declared version (1.26.4). Do not auto-update `go.mod` versions unless explicitly asked.

## Clean Architecture Convention

Software workshops follow a 3-layer variant: **Adapters / UseCases / Domain**. See `software/clean_arch.md` for the canonical definition.

- Interfaces defined in Domain, implemented in Adapters
- `context.Context` as first parameter everywhere
- No underscores in Go names (`MixedCaps` only)

## Container Runtime

Workshops use **Podman** (not Docker). Commands assume `podman` and `podman compose`. Some commands use `sudo podman` for port binding to privileged ports.

## Gotchas

- `make format` uses `pnpm dlx` if pnpm is available, otherwise `npx`. Both require network on first run.
- The `package.json` has `prettier@^3.8.3` — older prettier versions may behave differently.
- Infrastructure workshops with containers (`redis-up`, `mq-up`) need Podman running.
- Certificate files (`*.pem`, `*.key`, `*.crt`, `*.csr`) are git-ignored. Do not commit them.
- The `.ai-handoff.md` file is git-ignored but tracked in session protocol — read it on session start.
