package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/Dilgo-dev/tossit/internal/color"
	"github.com/Dilgo-dev/tossit/internal/transfer"
)

func runSend(args []string) {
	relayURL, relayToken, stream, _, paths := parseFlags(args)

	piped := stdinIsPipe()
	if len(paths) == 0 && !piped {
		fmt.Fprintln(os.Stderr, "Usage: tossit send [--relay URL] [--stream] <file|dir> ...")
		fmt.Fprintln(os.Stderr, "       echo \"text\" | tossit send")
		os.Exit(1)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
			os.Exit(1)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := transfer.SendOptions{
		RelayURL:   relayURL,
		RelayToken: relayToken,
		Paths:      paths,
		Stream:     stream,
	}

	if piped && len(paths) == 0 {
		opts.Stdin = os.Stdin
	}

	if err := transfer.Send(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
		os.Exit(1)
	}
}
