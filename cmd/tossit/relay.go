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
	var configPath string

	flagSet := map[string]bool{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Println("Usage: tossit relay [options]")
			fmt.Println()
			fmt.Println("Run a self-hosted relay server.")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --config <file>     Load config from JSON file")
			fmt.Println("  --port <port>       Port to listen on (default: 8080)")
			fmt.Println("  --storage <dir>     Storage directory (default: ./data)")
			fmt.Println("  --expire <duration> Transfer expiry (default: 24h)")
			fmt.Println("  --max-size <bytes>  Max file size per transfer (default: 2GB)")
			fmt.Println("  --rate-limit <n>    Max connections per minute per IP (default: 20, 0=off)")
			fmt.Println("  --auth-token <tok>  Require token for access (or set AUTH_TOKEN env)")
			fmt.Println("  --allow-ips <list>  Comma-separated IP allowlist")
			return
		case "--config":
			if i+1 < len(args) {
				i++
				configPath = args[i]
			}
		case "--port":
			if i+1 < len(args) {
				i++
				port = args[i]
				flagSet["port"] = true
			}
		case "--storage":
			if i+1 < len(args) {
				i++
				storageDir = args[i]
				flagSet["storage"] = true
			}
		case "--expire":
			if i+1 < len(args) {
				i++
				d, err := time.ParseDuration(args[i])
				if err == nil {
					expire = d
					flagSet["expire"] = true
				}
			}
		case "--max-size":
			if i+1 < len(args) {
				i++
				maxSize = relay.ParseSize(args[i])
				flagSet["max-size"] = true
			}
		case "--rate-limit":
			if i+1 < len(args) {
				i++
				n, err := strconv.Atoi(args[i])
				if err == nil {
					rateLimit = n
					flagSet["rate-limit"] = true
				}
			}
		case "--auth-token":
			if i+1 < len(args) {
				i++
				authToken = args[i]
				flagSet["auth-token"] = true
			}
		case "--allow-ips":
			if i+1 < len(args) {
				i++
				for _, ip := range strings.Split(args[i], ",") {
					if trimmed := strings.TrimSpace(ip); trimmed != "" {
						allowIPs = append(allowIPs, trimmed)
					}
				}
				flagSet["allow-ips"] = true
			}
		}
	}

	if configPath != "" {
		fc, err := relay.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
		if fc.Port != "" && !flagSet["port"] {
			port = fc.Port
		}
		if fc.Storage != "" && !flagSet["storage"] {
			storageDir = fc.Storage
		}
		if fc.Expire != "" && !flagSet["expire"] {
			if d, err := time.ParseDuration(fc.Expire); err == nil {
				expire = d
			}
		}
		if fc.MaxSize != "" && !flagSet["max-size"] {
			maxSize = relay.ParseSize(fc.MaxSize)
		}
		if fc.RateLimit != nil && !flagSet["rate-limit"] {
			rateLimit = *fc.RateLimit
		}
		if fc.AuthToken != "" && !flagSet["auth-token"] && os.Getenv("AUTH_TOKEN") == "" {
			authToken = fc.AuthToken
		}
		if len(fc.AllowIPs) > 0 && !flagSet["allow-ips"] {
			allowIPs = fc.AllowIPs
		}
	}

	cfg := relay.Config{
		Port:       port,
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
	http.HandleFunc("/health", r.HandleHealth)
	http.HandleFunc("/metrics", r.HandleMetrics)
	http.HandleFunc("/api/config", r.HandleConfig)
	http.Handle("/", r.WebHandler())

	addr := ":" + port
	log.Println("relay listening on " + addr + " (storage: " + storageDir + ", expire: " + expire.String() + ", max-size: " + relay.FormatSize(maxSize) + ")")
	srv := &http.Server{Addr: addr, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
