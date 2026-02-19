# Ginkgo Talk

AI phone keyboard for controlling text input on your PC over local network.

## Why Open Source

- Free and transparent by default
- Local-first design
- Easy for the community to audit and improve

## Features

- Pair phone with desktop via QR code + 4-digit pairing code
- Mobile input sends text to desktop target app
- One-tap send from phone (equivalent to desktop Enter)
- Desktop shortcuts: Enter, Shift+Enter, Clear, Undo, Tab, Paste, Esc
- Optional AI text processing modes: tidy, formal, translate
- Runtime AI configuration from mobile page
- Windows system tray with IP display, IP configuration, QR code, pair code, quit

## Tech Stack

- Go backend (`net/http` + `gorilla/websocket`)
- Static web app (HTML/CSS/JS)
- Local HTTPS + self-signed certificate generation
- Windows keyboard simulation via native calls

## Quick Start

### 1. Prerequisites

- Windows (required for keyboard simulation)
- Go 1.22+ recommended

### 2. Build & Run

```bash
# Build as GUI app (no console window)
go build -ldflags "-H windowsgui" -o GinkgoTalk.exe .

# Or simply run in development
go run .
```

You will see:

- Local URL
- QR-code endpoint (`/qrcode`)
- Pair code in terminal

### 3. Connect Phone

1. Open QR code page on desktop browser: `https://<LAN-IP>:9527/qrcode`
2. Scan with phone or manually open `https://<LAN-IP>:9527`
3. Enter the 4-digit pair code shown in terminal

## Optional AI Configuration

Set API key from mobile "AI settings", or via environment variables:

- `DEEPSEEK_API_KEY`
- `DEEPSEEK_BASE_URL` (optional)
- `DEEPSEEK_MODEL` (optional)

## Icon Replacement

If you want to replace app icons with your own image:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\generate_icons.ps1 -Source .\path\to\your-image.png
```

This will generate:

- `web/icon-512.png`
- `web/icon-192.png`
- `web/icon-32.png`

## Project Structure

```text
.
├── main.go                 # Entry point
├── server.go               # HTTP/WebSocket server, API handlers
├── ai.go                   # AI text processing (DeepSeek)
├── keyboard.go             # Windows keyboard simulation
├── config.go               # Persistent configuration
├── app_run_windows.go      # Windows system tray integration
├── app_run_default.go      # Non-Windows fallback
├── build.bat               # Windows build script
└── web/
    ├── index.html          # Mobile PWA page
    ├── app.js              # Frontend logic
    ├── style.css           # Styling
    └── manifest.json       # PWA manifest
```

## Security & Privacy

- Pairing required before accepting control commands
- Session token + device id checks on protected endpoints
- Intended for trusted LAN environments

If you discover a vulnerability, please follow `SECURITY.md`.

## Contributing

See `CONTRIBUTING.md`.

## License

MIT. See `LICENSE`.
