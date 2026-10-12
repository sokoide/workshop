# Workshop Document Template

All workshop documents (`infra/*.md`, `software/*.md`) follow this skeleton.
Gold-standard references: [`infra/cache_en.md`](../infra/cache_en.md) (compact hands-on)
and [`software/idempotency_en.md`](../software/idempotency_en.md) (full skeleton).

Every workshop document starts with the following block, immediately after the H1 title:

```markdown
> **⏱️ Estimated Time**: Approx. XX minutes
> **🎯 Level**: Beginner | Intermediate | Advanced
> **💻 Environment**: Linux (root/sudo) | Linux | macOS | any (containers only)

> **💡 Glossary**: For technical terms, see
> [Term 1](../infra/glossary_en.md#anchor), [Term 2](../infra/glossary_en.md#anchor)
```

Required sections, in this order:

| #   | Section               | Requirements                                                                                                                                                                         |
| --- | --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | `## Goal`             | What the learner can do afterwards. One diagram (mermaid) when the flow matters.                                                                                                     |
| 2   | `## Prerequisites`    | Checklist of tools **with versions**, OS requirements, cloud costs (if any), and prerequisite workshops (if any).                                                                    |
| 3   | `## Steps`            | `### STEP N: <imperative title>`. Each STEP: commands that are copy-pasteable, an `# Expected output:` line for every verification command, and a `### ✅ Checkpoint` checkbox list. |
| 4   | `## What you learned` | 3-6 bullet recap; links to related workshops as next steps.                                                                                                                          |
| 5   | `## Troubleshooting`  | Symptom → cause → fix, as a table or list.                                                                                                                                           |
| 6   | `## Cleanup`          | Commands to remove every container/VM/cloud resource created above.                                                                                                                  |
| 7   | `## References`       | Official docs only; 2-5 links.                                                                                                                                                       |

Rules:

- Bilingual pairs (`*_en.md` / `*_ja.md`) must keep identical heading structure (`make check-headings`).
- Every command block a reader cannot self-verify gets an expected-output comment.
- Go snippets follow repo conventions: `context.Context` first parameter, explicit error checks, MixedCaps.
- Keep single-session length under ~600 lines; split into parts beyond that and say so in the header block.

## Learning and Resume Conventions

- Start from the repository root; state the directory, terminal and host for command blocks.
- Match each asset's go.mod minimum; document extra tool requirements separately.
- Label code as a complete file, an edit, an explanatory excerpt, or a learner task. Link complete reference files for omitted implementations.
- Prepare isolated copies without overwriting learner work; state start and finish states for linked workshops.
- Place checkpoints after each behavior change. Include observable success/failure output and one prediction or explanation question with a hint/sample answer.
- For multiple sessions, specify a breakpoint, source/resources to retain, processes to stop, exact resume checks, and final cleanup. Separate reading/setup time from operations.
- Move optional background to a linked reference when it obscures the hands-on steps.
