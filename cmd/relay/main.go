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

var version = "dev"

func main() {
	port := "8080"
	storageDir := "./data"
	expire := 24 * time.Hour
	var maxSize int64 = 2 * 1024 * 1024 * 1024
	rateLimit := 20
	authToken := os.Getenv("AUTH_TOKEN")
	var allowIPs []string
	var configPath string
	uiEnabled := true
	uiPassword := ""
	adminPassword := ""

	flagSet := map[string]bool{}

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--version", "-v":
			fmt.Println("tossit-relay", version)
			return
		case "--help", "-h":
			fmt.Println("Usage: relay [options]")
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
			fmt.Println("  --ui <bool>         Enable web UI (default: true)")
			fmt.Println("  --ui-password <pw>  Password to access web UI")
			fmt.Println("  --admin-password <> Admin password (default: auto-generated, 'off' to disable)")
			fmt.Println("  --version           Show version")
			return
		case "--config":
			if i+1 < len(os.Args) {
				i++
				configPath = os.Args[i]
			}
		case "--port":
			if i+1 < len(os.Args) {
				i++
				port = os.Args[i]
				flagSet["port"] = true
			}
		case "--storage":
			if i+1 < len(os.Args) {
				i++
				storageDir = os.Args[i]
				flagSet["storage"] = true
			}
		case "--expire":
			if i+1 < len(os.Args) {
				i++
				d, err := time.ParseDuration(os.Args[i])
				if err == nil {
					expire = d
					flagSet["expire"] = true
				}
			}
		case "--max-size":
			if i+1 < len(os.Args) {
				i++
				maxSize = relay.ParseSize(os.Args[i])
				flagSet["max-size"] = true
			}
		case "--rate-limit":
			if i+1 < len(os.Args) {
				i++
				n, err := strconv.Atoi(os.Args[i])
				if err == nil {
					rateLimit = n
					flagSet["rate-limit"] = true
				}
			}
		case "--auth-token":
			if i+1 < len(os.Args) {
				i++
				authToken = os.Args[i]
				flagSet["auth-token"] = true
			}
		case "--allow-ips":
			if i+1 < len(os.Args) {
				i++
				for _, ip := range strings.Split(os.Args[i], ",") {
					if trimmed := strings.TrimSpace(ip); trimmed != "" {
						allowIPs = append(allowIPs, trimmed)
					}
				}
				flagSet["allow-ips"] = true
			}
		case "--ui":
			if i+1 < len(os.Args) {
				i++
				uiEnabled = os.Args[i] != "false"
				flagSet["ui"] = true
			}
		case "--ui-password":
			if i+1 < len(os.Args) {
				i++
				uiPassword = os.Args[i]
				flagSet["ui-password"] = true
			}
		case "--admin-password":
			if i+1 < len(os.Args) {
				i++
				adminPassword = os.Args[i]
				flagSet["admin-password"] = true
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
		if fc.UIEnabled != nil && !flagSet["ui"] {
			uiEnabled = *fc.UIEnabled
		}
		if fc.UIPassword != nil && !flagSet["ui-password"] {
			uiPassword = *fc.UIPassword
			flagSet["ui-password"] = true
		}
		if fc.AdminPassword != "" && !flagSet["admin-password"] {
			adminPassword = fc.AdminPassword
		}
	}

	cfg := relay.Config{
		Port:          port,
		StorageDir:    storageDir,
		Expire:        expire,
		MaxSize:       maxSize,
		Version:       version,
		RateLimit:     rateLimit,
		AuthToken:     authToken,
		AllowIPs:      allowIPs,
		UIEnabled:     uiEnabled,
		UIPassword:    uiPassword,
		UIPasswordSet: flagSet["ui-password"],
		AdminPassword: adminPassword,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	r := relay.New(cfg)
	go r.StartCleanup(ctx)

	http.HandleFunc("/ws", r.HandleConn)
	http.HandleFunc("/health", r.HandleHealth)
	http.HandleFunc("/metrics", r.HandleMetrics)
	http.HandleFunc("/api/config", r.HandleConfig)
	http.HandleFunc("/api/login", r.HandleLogin)
	http.HandleFunc("/api/login/admin", r.HandleAdminLogin)
	http.Handle("/", r.WebHandler())

	addr := ":" + port
	log.Println("relay listening on", addr) //nolint:gosec // log message from trusted CLI flags
	srv := &http.Server{Addr: addr, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
