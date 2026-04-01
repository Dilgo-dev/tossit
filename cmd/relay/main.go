package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("tossit-relay", version)
		return
	}

	fmt.Println("tossit-relay", version)
}
