package main

const defaultRelay = "wss://relay.tossit.dev/ws"

func parseFlags(args []string) (relayURL string, relayToken string, stream bool, dir string, password string, expires string, direct bool, stunServer string, multi string, remaining []string) {
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
		case "--direct":
			direct = true
		case "--stun":
			if i+1 < len(args) {
				stunServer = args[i+1]
				i++
			}
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
		case "--expires", "-e":
			if i+1 < len(args) {
				expires = args[i+1]
				i++
			}
		case "--multi", "-m":
			if i+1 < len(args) {
				multi = args[i+1]
				i++
			}
		default:
			remaining = append(remaining, args[i])
		}
	}
	return
}
