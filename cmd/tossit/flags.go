package main

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultRelay = "wss://relay.tossit.dev/ws"

func parseFlags(args []string) (relayURL string, relayToken string, stream bool, dir string, password string, expires string, direct bool, stunServer string, multi string, approve bool, limit string, remaining []string) {
	relayURL = defaultRelay
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--relay":
			if i+1 < len(args) {
				relayURL = args[i+1]
				i++
			}
		case "--relay-token":
			if i+1 < len(args) {
				relayToken = args[i+1]
				i++
			}
		case "--stream":
			stream = true
		case "--direct":
			direct = true
		case "--stun":
			if i+1 < len(args) {
				stunServer = args[i+1]
				i++
			}
		case "--dir", "-d":
			if i+1 < len(args) {
				dir = args[i+1]
				i++
			}
		case "--password", "-p":
			if i+1 < len(args) {
				password = args[i+1]
				i++
			}
		case "--expires", "-e":
			if i+1 < len(args) {
				expires = args[i+1]
				i++
			}
		case "--multi", "-m":
			if i+1 < len(args) {
				multi = args[i+1]
				i++
			}
		case "--approve":
			approve = true
		case "--limit", "-l":
			if i+1 < len(args) {
				limit = args[i+1]
				i++
			}
		default:
			remaining = append(remaining, args[i])
		}
	}
	return
}

func parseLimit(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "/s")
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)

	multipliers := []struct {
		suffix string
		mult   int64
	}{
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, m := range multipliers {
		if strings.HasSuffix(s, m.suffix) {
			numStr := strings.TrimSpace(strings.TrimSuffix(s, m.suffix))
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", numStr)
			}
			if n <= 0 {
				return 0, fmt.Errorf("limit must be positive")
			}
			return int64(n * float64(m.mult)), nil
		}
	}

	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid limit format: %s", s)
	}
	if n <= 0 {
		return 0, fmt.Errorf("limit must be positive")
	}
	return int64(n), nil
}
