package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/Dilgo-dev/tossit/internal/color"
	"github.com/Dilgo-dev/tossit/internal/transfer"
)

func runSend(args []string) {
	relayURL, relayToken, stream, _, password, expires, direct, stunServer, multi, paths := parseFlags(args)

	piped := stdinIsPipe()
	if len(paths) == 0 && !piped {
		fmt.Fprintln(os.Stderr, "Usage: tossit send [--relay URL] [--stream] [--password PW] <file|dir> ...")
		fmt.Fprintln(os.Stderr, "       echo \"text\" | tossit send")
		os.Exit(1)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil { //nolint:gosec // CLI receives paths from user args
			fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
			os.Exit(1)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var expireDuration time.Duration
	if expires != "" {
		d, err := time.ParseDuration(expires)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s invalid --expires duration: %s\n", color.BoldRed("Error:"), err)
			os.Exit(1)
		}
		expireDuration = d
	}

	var multiCount int
	if multi != "" {
		n, err := strconv.Atoi(multi)
		if err != nil || n < 1 {
			fmt.Fprintf(os.Stderr, "%s --multi must be a positive integer\n", color.BoldRed("Error:"))
			os.Exit(1)
		}
		multiCount = n
	}

	if direct {
		stream = true
	}

	opts := transfer.SendOptions{
		RelayURL:   relayURL,
		RelayToken: relayToken,
		Paths:      paths,
		Stream:     stream,
		Password:   password,
		Expires:    expireDuration,
		Direct:     direct,
		StunServer: stunServer,
		Multi:      multiCount,
	}

	if piped && len(paths) == 0 {
		opts.Stdin = os.Stdin
	}

	if err := transfer.Send(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
		os.Exit(1)
	}
}
