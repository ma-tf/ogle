---
name: commit-msg
description: Generate conventional commit messages from staged changes. Use when user says "write a commit", "commit message", "commit changes", or asks to generate a commit.
---

# commit-msg

Generates a conventional commit message from `git diff --cached`. Checks for commit linting rules if the project has them.

## Steps

1. Read staged diff
2. Write conventional commit message (see format below)
3. Execute `.git/hooks/commit-msg` against the message if the hook exists (read-only — safe to run)
4. Print it — don't stage, amend, or commit

## Conventional commit format

```text
<type>(<scope>): <subject>

<body>
```

`type` — one of: `feat`, `fix`, `chore`, `refactor`, `docs`, `style`, `test`, `perf`, `build`, `ci`, `revert`.

`scope` — the module or area affected (optional). Infer from the paths in the staged diff.

`subject` — imperative tense, no capital, no period. Aim for ≤50 chars, hard limit 72.

`body` — optional. Explain what and why, not how. Wrap at 72 chars.

Use `BREAKING CHANGE:` trailer or `!` after type/scope for breaking changes.
