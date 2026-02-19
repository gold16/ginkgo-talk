# Release Checklist

## Pre-release

- [ ] `go build ./...` passes
- [ ] Manual end-to-end test on LAN passes
- [ ] Pairing flow works (QR + manual URL + pair code)
- [ ] One-tap send from phone triggers desktop send
- [ ] AI modes verified (if enabled)
- [ ] Docs updated (`README.md`, changelog/release notes)

## Packaging

- [ ] Build release binary
- [ ] Verify no local secrets/artifacts are included
- [ ] Tag version in git (e.g. `v0.1.1`)

## GitHub Release

- [ ] Create release notes (features, fixes, known issues)
- [ ] Upload binaries/assets
- [ ] Link migration notes if needed

