package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/Dilgo-dev/tossit/internal/transfer"
)

func runReceive(args []string) {
	relayURL, relayToken, _, remaining := parseFlags(args)
	if len(remaining) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: tossit receive [--relay URL] [--relay-token TOKEN] <code>")
		os.Exit(1)
	}

	code := remaining[0]
	outputDir := "."
	if len(remaining) > 1 {
		outputDir = remaining[1]
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := transfer.ReceiveOptions{
		RelayURL:   relayURL,
		RelayToken: relayToken,
		Code:       code,
		OutputDir:  outputDir,
	}

	if err := transfer.Receive(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
