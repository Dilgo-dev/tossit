package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/Dilgo-dev/tossit/internal/transfer"
)

func runSend(args []string) {
	relayURL, paths := parseFlags(args)
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: tossit send [--relay URL] <file|dir> ...")
		os.Exit(1)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := transfer.SendOptions{
		RelayURL: relayURL,
		Paths:    paths,
	}

	if err := transfer.Send(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
