# Security Policy

## Supported Scope

This project is intended for trusted local networks (LAN).

Security-sensitive areas include:

- Pairing and authorization flow
- WebSocket command handling
- Keyboard input simulation layer

## Reporting a Vulnerability

Please do **not** open a public issue for unpatched vulnerabilities.

Report privately with:

- Affected version/commit
- Reproduction steps
- Impact assessment
- Suggested mitigation (if available)

Maintainers will acknowledge receipt and provide a remediation timeline.

Contact: [GitHub Security Advisories](https://github.com/gold16/ginkgo-talk/security/advisories/new) or email <xrgold16@outlook.com>

## Hardening Recommendations

- Run only on trusted LAN
- Rotate sessions by restarting server when needed
- Do not expose service directly to public internet
