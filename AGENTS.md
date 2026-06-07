# AGENTS.md

AI agent conventions supplementing `CLAUDE.md`. Read both on session start.

## Markdown Formatting

```bash
make format
```

Runs `prettier --write --parser markdown` → `markdownlint --fix` → `textlint --fix`. Requires `node_modules/` in the root (`pnpm install` or `npm install` first).

**Excluded from formatting:**

- `CLAUDE.md`
- `conductor/`, `node_modules/`, `.gomodcache/` (entire directories skipped by `make format`)

markdownlint disables line-length, duplicate headings, inline HTML, and emphasis-as-heading rules. textlint uses `preset-ja-spacing`.

## Bilingual Files

Every workshop has `_ja.md` and `_en.md` pairs. They must stay in sync structurally — editing one requires updating the other. The root `README.md` (English) and `README_ja.md` (Japanese) are the indexes.

## Conventions Not in CLAUDE.md

- **Do not auto-update `go.mod` versions** — they may intentionally lag behind the repo-declared version.
- **Do not reformat `CLAUDE.md`** — it is excluded from `make format`.
- **Do not commit certificate files** (`*.pem`, `*.key`, `*.crt`, `*.csr`) — they are git-ignored.

## Gotchas

- `make format` uses `pnpm dlx` if available, otherwise `npx`. Both need network on first run.
- `package.json` pins `prettier@^3.8.3` — older prettier versions may behave differently.
- Infrastructure workshops with containers (`redis-up`, `mq-up`) need Podman running.
- `.ai-handoff.md` is git-ignored but required by session protocol — read it on session start.
