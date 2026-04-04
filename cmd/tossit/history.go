package main

import (
	"fmt"
	"os"

	"github.com/Dilgo-dev/tossit/internal/color"
	"github.com/Dilgo-dev/tossit/internal/history"
	"github.com/Dilgo-dev/tossit/internal/progress"
)

func runHistory(args []string) {
	if len(args) > 0 && args[0] == "clear" {
		if err := history.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
			os.Exit(1)
		}
		fmt.Println(color.Green("History cleared."))
		return
	}

	entries, err := history.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", color.BoldRed("Error:"), err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println(color.Dim("No transfer history."))
		return
	}

	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		arrow := color.Cyan("->")
		dir := color.Bold("sent")
		if e.Direction == history.Received {
			arrow = color.Green("<-")
			dir = color.Bold("received")
		}

		size := progress.FormatSize(e.Size)
		ts := e.Time.Local().Format("2006-01-02 15:04")

		fmt.Printf("  %s %s %s %s %s %s\n",
			arrow,
			dir,
			color.Bold(e.Name),
			color.Dim("("+size+")"),
			color.Dim("code:"+e.Code),
			color.Dim(ts),
		)
	}
}
