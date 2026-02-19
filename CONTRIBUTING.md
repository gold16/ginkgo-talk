# Contributing Guide

Thanks for contributing to Ginkgo Talk.

## Before You Start

- Read `README.md` and `SECURITY.md`
- Search existing issues/PRs to avoid duplicates
- Keep changes focused and minimal

## Development Setup

```bash
go run .
go test ./...
```

## Branch & Commit

- Use short feature branches, for example:
  - `feat/mobile-pairing`
  - `fix/ws-timeout`
- Write clear commit messages:
  - `feat: one-tap send triggers desktop enter`
  - `fix: handle pair timeout state`

## Pull Request Checklist

- Code builds successfully
- Behavior is tested manually (and with tests when possible)
- No unrelated file changes included
- Docs updated if user-facing behavior changed

## Coding Notes

- Prefer small, reviewable PRs
- Preserve backward compatibility where possible
- Do not commit local runtime artifacts (`cert.pem`, `key.pem`, `gtalk_config.json`, binaries)

