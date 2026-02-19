package main

import (
	"fmt"
	"log"
)

const (
	defaultPort = ":9527"
	appName     = "Ginkgo Talk"
	appVersion  = "0.1.0"
)

func main() {
	if err := runApp(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func printStartupInfo(server *Server) {
	fmt.Printf("%s v%s - AI mobile keyboard\n", appName, appVersion)
	lanIP := server.LanIP()
	log.Printf("Local IP: %s", lanIP)
	log.Printf("URL: https://%s%s", lanIP, defaultPort)
	log.Printf("Open https://%s%s/qrcode in browser to see QR code", lanIP, defaultPort)
	fmt.Println()
}
