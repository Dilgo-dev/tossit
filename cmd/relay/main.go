package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Dilgo-dev/tossit/internal/relay"
)

var version = "dev"

func main() {
	port := "8080"

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--version", "-v":
			fmt.Println("tossit-relay", version)
			return
		case "--help", "-h":
			fmt.Println("Usage: relay [--port PORT]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --port    Port to listen on (default: 8080)")
			fmt.Println("  --version Show version")
			return
		case "--port":
			if i+1 < len(os.Args) {
				i++
				port = os.Args[i]
			}
		}
	}

	r := relay.New()
	http.HandleFunc("/ws", r.HandleConn)

	log.Printf("relay listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
