package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/Dilgo-dev/tossit/internal/relay"
)

func runRelay(args []string) {
	port := "8080"
	storageDir := "./data"
	expire := 24 * time.Hour
	var maxSize int64 = 2 * 1024 * 1024 * 1024
	rateLimit := 20
	authToken := os.Getenv("AUTH_TOKEN")
	var allowIPs []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Println("Usage: tossit relay [options]")
			fmt.Println()
			fmt.Println("Run a self-hosted relay server.")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --port <port>       Port to listen on (default: 8080)")
			fmt.Println("  --storage <dir>     Storage directory (default: ./data)")
			fmt.Println("  --expire <duration> Transfer expiry (default: 24h)")
			fmt.Println("  --max-size <bytes>  Max file size per transfer (default: 2GB)")
			fmt.Println("  --rate-limit <n>    Max connections per minute per IP (default: 20, 0=off)")
			fmt.Println("  --auth-token <tok>  Require token for access (or set AUTH_TOKEN env)")
			fmt.Println("  --allow-ips <list>  Comma-separated IP allowlist")
			return
		case "--port":
			if i+1 < len(args) {
				i++
				port = args[i]
			}
		case "--storage":
			if i+1 < len(args) {
				i++
				storageDir = args[i]
			}
		case "--expire":
			if i+1 < len(args) {
				i++
				d, err := time.ParseDuration(args[i])
				if err == nil {
					expire = d
				}
			}
		case "--max-size":
			if i+1 < len(args) {
				i++
				maxSize = parseRelaySize(args[i])
			}
		case "--rate-limit":
			if i+1 < len(args) {
				i++
				n, err := strconv.Atoi(args[i])
				if err == nil {
					rateLimit = n
				}
			}
		case "--auth-token":
			if i+1 < len(args) {
				i++
				authToken = args[i]
			}
		case "--allow-ips":
			if i+1 < len(args) {
				i++
				for _, ip := range strings.Split(args[i], ",") {
					if trimmed := strings.TrimSpace(ip); trimmed != "" {
						allowIPs = append(allowIPs, trimmed)
					}
				}
			}
		}
	}

	cfg := relay.Config{
		StorageDir: storageDir,
		Expire:     expire,
		MaxSize:    maxSize,
		Version:    version,
		RateLimit:  rateLimit,
		AuthToken:  authToken,
		AllowIPs:   allowIPs,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	r := relay.New(cfg)
	go r.StartCleanup(ctx)

	http.HandleFunc("/ws", r.HandleConn)
	http.HandleFunc("/d/", r.HandleWeb)
	http.HandleFunc("/health", r.HandleHealth)
	http.HandleFunc("/metrics", r.HandleMetrics)

	log.Printf("relay listening on :%s (storage: %s, expire: %s, max-size: %s)",
		port, storageDir, expire, formatRelaySize(maxSize))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func parseRelaySize(s string) int64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	multiplier := int64(1)
	if strings.HasSuffix(s, "GB") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "KB") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "KB")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 2 * 1024 * 1024 * 1024
	}
	return n * multiplier
}

func formatRelaySize(bytes int64) string {
	switch {
	case bytes >= 1024*1024*1024:
		return fmt.Sprintf("%dGB", bytes/(1024*1024*1024))
	case bytes >= 1024*1024:
		return fmt.Sprintf("%dMB", bytes/(1024*1024))
	default:
		return fmt.Sprintf("%dKB", bytes/1024)
	}
}
