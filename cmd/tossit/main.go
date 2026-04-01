package main

import (
	"fmt"
	"os"

	"github.com/Dilgo-dev/tossit/internal/update"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "send", "s":
		runSend(os.Args[2:])
	case "receive", "recv", "r":
		runReceive(os.Args[2:])
	case "relay":
		runRelay(os.Args[2:])
	case "update":
		if err := update.Run(version); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	case "--version", "-v":
		fmt.Println("tossit", version)
	case "--help", "-h", "help":
		printHelp()
	default:
		runSend(os.Args[1:])
	}
}

func printHelp() {
	fmt.Printf("tossit %s - file transfer tool\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  tossit <file|dir> ...         Upload and share files")
	fmt.Println("  tossit send <file|dir> ...    Same as above (explicit)")
	fmt.Println("  tossit receive <code>         Download files")
	fmt.Println("  tossit relay                 Run a self-hosted relay server")
	fmt.Println("  tossit update                Check for updates")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --relay <url>      Relay server URL (default: wss://relay.tossit.dev/ws)")
	fmt.Println("  --relay-token <t>  Auth token for private relay servers")
	fmt.Println("  --stream           Real-time streaming (both sides must be online)")
	fmt.Println("  --version          Show version")
	fmt.Println("  --help             Show this help")
}
