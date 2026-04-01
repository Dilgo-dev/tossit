package qr

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

func Print(url string) {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return
	}

	bitmap := q.Bitmap()
	size := len(bitmap)

	fmt.Println()
	for y := 0; y < size-1; y += 2 {
		fmt.Print("  ")
		for x := 0; x < size; x++ {
			top := bitmap[y][x]
			bottom := bitmap[y+1][x]

			switch {
			case top && bottom:
				fmt.Print("\u2588")
			case top && !bottom:
				fmt.Print("\u2580")
			case !top && bottom:
				fmt.Print("\u2584")
			default:
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}

	if size%2 == 1 {
		fmt.Print("  ")
		for x := 0; x < size; x++ {
			if bitmap[size-1][x] {
				fmt.Print("\u2580")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}
	fmt.Println()
}
