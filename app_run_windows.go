//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"unsafe"

	"github.com/getlantern/systray"
)

//go:embed web/tray-icon.ico
var trayIcon []byte

func hideConsoleWindow() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	procShowWindow := user32.NewProc("ShowWindow")

	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, 0) // SW_HIDE = 0
	}
}

func runApp() error {
	hideConsoleWindow()

	server := NewServer(defaultPort)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	resultCh := make(chan error, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	systray.Run(func() {
		setupTray(server, serverErrCh, sigCh, resultCh)
	}, func() {})

	err := <-resultCh
	return err
}

func setupTray(server *Server, serverErrCh <-chan error, sigCh <-chan os.Signal, resultCh chan<- error) {
	systray.SetTitle(appName)
	systray.SetTooltip("Ginkgo Talk")
	if len(trayIcon) > 0 {
		systray.SetIcon(trayIcon)
	}

	lanIP := server.LanIP()
	qrURL := fmt.Sprintf("https://%s%s/qrcode", lanIP, defaultPort)

	ipItem := systray.AddMenuItem(fmt.Sprintf("IP: %s%s", lanIP, defaultPort), "Current server address")
	ipItem.Disable()
	setIPItem := systray.AddMenuItem("Set IP...", "Set LAN IP address")
	systray.AddSeparator()
	openQRItem := systray.AddMenuItem("Open QR Code", "Open QR code page in browser")
	pairCodeItem := systray.AddMenuItem(fmt.Sprintf("Pair Code: %s", server.pairCode), "Current pair code")
	pairCodeItem.Disable()
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "Quit Ginkgo Talk")

	go func() {
		for {
			select {
			case <-setIPItem.ClickedCh:
				newIP := showIPInputDialog(server.LanIP())
				if newIP != "" {
					applyIPChange(server, newIP, ipItem)
				}
			case <-openQRItem.ClickedCh:
				curIP := server.LanIP()
				qrURL = fmt.Sprintf("https://%s%s/qrcode", curIP, defaultPort)
				openBrowser(qrURL)
			case <-quitItem.ClickedCh:
				resultCh <- nil
				systray.Quit()
				return
			case <-sigCh:
				resultCh <- nil
				systray.Quit()
				return
			case err := <-serverErrCh:
				resultCh <- err
				systray.Quit()
				return
			}
		}
	}()
}

// showIPInputDialog shows a Windows input dialog for IP address configuration.
// Returns the user input string, or "" if cancelled.
func showIPInputDialog(currentIP string) string {
	psScript := fmt.Sprintf(`
Add-Type -AssemblyName Microsoft.VisualBasic
$result = [Microsoft.VisualBasic.Interaction]::InputBox(
    "Enter LAN IP address (or 'auto' for auto-detect):",
    "Ginkgo Talk - Set IP",
    "%s"
)
if ($result -ne "") { Write-Output $result }
`, currentIP)

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("IP input dialog error: %v", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

// applyIPChange validates and applies a new IP, updates tray display and saves config.
func applyIPChange(server *Server, input string, ipItem *systray.MenuItem) {
	if strings.EqualFold(input, "auto") {
		server.SetLanIPOverride("")
		newIP := server.LanIP()
		ipItem.SetTitle(fmt.Sprintf("IP: %s%s", newIP, defaultPort))
		log.Printf("LAN IP reset to auto-detect: %s", newIP)
	} else {
		ip := net.ParseIP(strings.TrimSpace(input))
		if ip == nil || ip.To4() == nil {
			log.Printf("Invalid IP from dialog: %s", input)
			showErrorDialog("Invalid IP address: " + input)
			return
		}
		server.SetLanIPOverride(ip.String())
		ipItem.SetTitle(fmt.Sprintf("IP: %s%s", ip.String(), defaultPort))
		log.Printf("LAN IP set to: %s", ip.String())
	}
	// Persist to config
	cfg := LoadConfig()
	cfg.LanIP = server.GetLanIPOverride()
	SaveConfig(cfg)
}

// showErrorDialog shows a simple Windows error message box.
func showErrorDialog(msg string) {
	msgBox := user32.NewProc("MessageBoxW")
	title, _ := syscall.UTF16PtrFromString("Ginkgo Talk")
	text, _ := syscall.UTF16PtrFromString(msg)
	msgBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x10) // MB_ICONERROR
}

func openBrowser(rawURL string) {
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		log.Printf("invalid URL: %s", rawURL)
		return
	}
	if err := exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start(); err != nil {
		log.Printf("open browser failed: %v", err)
	}
}
