package main

const defaultRelay = "wss://relay.tossit.dev/ws"

func parseFlags(args []string) (relayURL string, remaining []string) {
	relayURL = defaultRelay
	for i := 0; i < len(args); i++ {
		if args[i] == "--relay" && i+1 < len(args) {
			relayURL = args[i+1]
			i++
		} else {
			remaining = append(remaining, args[i])
		}
	}
	return
}
