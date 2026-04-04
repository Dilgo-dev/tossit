package main

const defaultRelay = "wss://relay.tossit.dev/ws"

func parseFlags(args []string) (relayURL string, relayToken string, stream bool, dir string, password string, remaining []string) {
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
		default:
			remaining = append(remaining, args[i])
		}
	}
	return
}
