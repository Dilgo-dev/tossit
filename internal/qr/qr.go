package qr

import (
	"fmt"
	"strings"

	"github.com/skip2/go-qrcode"
)

const (
	qrWhite = "\033[97m" // bright white foreground
	qrBlack = "\033[40m" // black background
	qrReset = "\033[0m"
)

func Print(url string) {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return
	}

	bitmap := q.Bitmap()
	size := len(bitmap)

	fmt.Println()

	// Quiet zone top: full white line
	quietLine := qrBlack + qrWhite + "  "
	quietLine += strings.Repeat("\u2588", size+4)
	quietLine += "  " + qrReset
	fmt.Println(quietLine)

	for y := 0; y < size-1; y += 2 {
		var line strings.Builder
		line.WriteString(qrBlack + qrWhite + "  \u2588\u2588")
		for x := 0; x < size; x++ {
			top := bitmap[y][x]
			bottom := bitmap[y+1][x]

			// Inverted: dark modules = space (background), light modules = block
			switch {
			case top && bottom:
				line.WriteString(" ")
			case top && !bottom:
				line.WriteString("\u2584")
			case !top && bottom:
				line.WriteString("\u2580")
			default:
				line.WriteString("\u2588")
			}
		}
		line.WriteString("\u2588\u2588  " + qrReset)
		fmt.Println(line.String())
	}

	if size%2 == 1 {
		var line strings.Builder
		line.WriteString(qrBlack + qrWhite + "  \u2588\u2588")
		for x := 0; x < size; x++ {
			if bitmap[size-1][x] {
				line.WriteString("\u2584")
			} else {
				line.WriteString("\u2588")
			}
		}
		line.WriteString("\u2588\u2588  " + qrReset)
		fmt.Println(line.String())
	}

	// Quiet zone bottom
	fmt.Println(quietLine)
	fmt.Println()
}
