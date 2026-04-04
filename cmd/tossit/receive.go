package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/Dilgo-dev/tossit/internal/color"
	"github.com/Dilgo-dev/tossit/internal/transfer"
)

func runReceive(args []string) {
	relayURL, relayToken, _, dir, password, remaining := parseFlags(args)
	if len(remaining) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: tossit receive [--relay URL] [--dir PATH] [--password PW] <code>")
		os.Exit(1)
	}

	code := remaining[0]
	outputDir := "."
	if dir != "" {
		outputDir = dir
	} else if len(remaining) > 1 {
		outputDir = remaining[1]
	}

	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0o750); err != nil { //nolint:gosec // CLI receives dir from user args
			fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
			os.Exit(1)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := transfer.ReceiveOptions{
		RelayURL:   relayURL,
		RelayToken: relayToken,
		Code:       code,
		OutputDir:  outputDir,
		Password:   password,
	}

	if err := transfer.Receive(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
		os.Exit(1)
	}
}
