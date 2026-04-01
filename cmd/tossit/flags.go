package main

const defaultRelay = "wss://relay.tossit.dev/ws"

func parseFlags(args []string) (relayURL string, stream bool, remaining []string) {
	relayURL = defaultRelay
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--relay":
			if i+1 < len(args) {
				relayURL = args[i+1]
				i++
			}
		case "--stream":
			stream = true
		default:
			remaining = append(remaining, args[i])
		}
	}
	return
}
