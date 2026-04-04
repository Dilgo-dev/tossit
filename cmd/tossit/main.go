package main

import (
	"fmt"
	"os"

	"github.com/Dilgo-dev/tossit/internal/color"
	"github.com/Dilgo-dev/tossit/internal/update"
)

var version = "dev"

const banner = `  ______                   ______     __
 /_  __/___  __________   /  _/ /_   / /
  / / / __ \/ ___/ ___/   / // __/  / /
 / / / /_/ (__  |__  )  _/ // /_   /_/
/_/  \____/____/____/  /___/\__/  (_)   `

func main() {
	if len(os.Args) < 2 {
		if stdinIsPipe() {
			runSend(nil)
			return
		}
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
	case "history":
		runHistory(os.Args[2:])
	case "completion":
		runCompletion(os.Args[2:])
	case "update":
		if err := update.Run(version); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
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
	fmt.Println(color.Green(banner))
	fmt.Println()
	fmt.Printf("  %s %s\n\n", color.Bold("tossit"), color.Dim(version))
	fmt.Println(color.Bold("Usage:"))
	fmt.Printf("  %s         Upload and share files\n", color.Cyan("tossit <file|dir> ..."))
	fmt.Printf("  %s    Same as above (explicit)\n", color.Cyan("tossit send <file|dir> ..."))
	fmt.Printf("  %s         Download files\n", color.Cyan("tossit receive <code>"))
	fmt.Printf("  %s                 Run a self-hosted relay server\n", color.Cyan("tossit relay"))
	fmt.Printf("  %s               Show transfer history\n", color.Cyan("tossit history"))
	fmt.Printf("  %s                Check for updates\n", color.Cyan("tossit update"))
	fmt.Printf("  %s  Generate shell completions\n", color.Cyan("tossit completion <shell>"))
	fmt.Println()
	fmt.Println(color.Bold("Options:"))
	fmt.Printf("  %s      Relay server URL\n", color.Yellow("--relay <url>"))
	fmt.Printf("  %s  Auth token for private relay\n", color.Yellow("--relay-token <t>"))
	fmt.Printf("  %s           Real-time streaming (both sides online)\n", color.Yellow("--stream"))
	fmt.Printf("  %s           Direct P2P transfer with NAT traversal\n", color.Yellow("--direct"))
	fmt.Printf("  %s     Custom STUN server (default: stun.l.google.com:19302)\n", color.Yellow("--stun <addr>"))
	fmt.Printf("  %s    Save files to directory (receive only)\n", color.Yellow("--dir <path>"))
	fmt.Printf("  %s  Password-protect the transfer\n", color.Yellow("--password <pw>"))
	fmt.Printf("  %s  Transfer expiry (e.g. 1h, 30m, 12h)\n", color.Yellow("--expires <dur>"))
	fmt.Printf("  %s      Allow N downloads from the same code\n", color.Yellow("--multi <N>"))
	fmt.Printf("  %s          Approve each download before it starts\n", color.Yellow("--approve"))
	fmt.Printf("  %s  Throttle speed (e.g. 1MB/s, 500KB/s)\n", color.Yellow("--limit <rate>"))
	fmt.Printf("  %s           Show version\n", color.Yellow("--version"))
	fmt.Printf("  %s              Show this help\n", color.Yellow("--help"))
	fmt.Println()
	fmt.Println(color.Bold("Pipe:"))
	fmt.Printf("  %s\n", color.Dim("echo \"hello\" | tossit"))
	fmt.Printf("  %s\n", color.Dim("cat file.sql | tossit send"))
}

func stdinIsPipe() bool {
	fi, _ := os.Stdin.Stat()
	return fi.Mode()&os.ModeCharDevice == 0
}
