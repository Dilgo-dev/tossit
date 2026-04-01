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

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--version", "-v":
			fmt.Println("tossit-relay", version)
			return
		case "--help", "-h":
			fmt.Println("Usage: relay [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --port <port>       Port to listen on (default: 8080)")
			fmt.Println("  --storage <dir>     Storage directory (default: ./data)")
			fmt.Println("  --expire <duration> Transfer expiry (default: 24h)")
			fmt.Println("  --max-size <bytes>  Max file size per transfer (default: 2GB)")
			fmt.Println("  --version           Show version")
			return
		case "--port":
			if i+1 < len(os.Args) {
				i++
				port = os.Args[i]
			}
		case "--storage":
			if i+1 < len(os.Args) {
				i++
				storageDir = os.Args[i]
			}
		case "--expire":
			if i+1 < len(os.Args) {
				i++
				d, err := time.ParseDuration(os.Args[i])
				if err == nil {
					expire = d
				}
			}
		case "--max-size":
			if i+1 < len(os.Args) {
				i++
				maxSize = parseSize(os.Args[i])
			}
		}
	}

	cfg := relay.Config{
		StorageDir: storageDir,
		Expire:     expire,
		MaxSize:    maxSize,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	r := relay.New(cfg)
	go r.StartCleanup(ctx)

	http.HandleFunc("/ws", r.HandleConn)
	http.HandleFunc("/d/", r.HandleWeb)

	log.Printf("relay listening on :%s (storage: %s, expire: %s, max-size: %s)",
		port, storageDir, expire, formatSize(maxSize))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func parseSize(s string) int64 {
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

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024*1024:
		return fmt.Sprintf("%dGB", bytes/(1024*1024*1024))
	case bytes >= 1024*1024:
		return fmt.Sprintf("%dMB", bytes/(1024*1024))
	default:
		return fmt.Sprintf("%dKB", bytes/1024)
	}
}
