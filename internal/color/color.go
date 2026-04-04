package color

import (
	"fmt"
	"os"
)

var enabled = func() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	o, _ := os.Stdout.Stat()
	return o.Mode()&os.ModeCharDevice != 0
}()

const (
	reset = "\033[0m"
	bold  = "\033[1m"
	dim   = "\033[2m"

	// tossit.dev palette
	accent    = "\033[38;2;163;230;53m"  // #A3E635 lime green
	accentDim = "\033[38;2;101;163;13m"  // #65A30D
	success   = "\033[38;2;52;211;153m"  // #34d399 emerald-400
	textDim   = "\033[38;2;139;139;150m" // #8b8b96
	red       = "\033[38;2;255;95;87m"   // #ff5f57
)

func wrap(style, s string) string {
	if !enabled {
		return s
	}
	return style + s + reset
}

func Bold(s string) string      { return wrap(bold, s) }
func Dim(s string) string       { return wrap(textDim, s) }
func Red(s string) string       { return wrap(red, s) }
func Green(s string) string     { return wrap(success, s) }
func Yellow(s string) string    { return wrap(accentDim, s) }
func Cyan(s string) string      { return wrap(accent, s) }
func BoldCyan(s string) string  { return wrap(bold+accent, s) }
func BoldRed(s string) string   { return wrap(bold+red, s) }
func Accent(s string) string    { return wrap(accent, s) }
func AccentDim(s string) string { return wrap(accentDim, s) }

func Sprintf(style, format string, a ...any) string {
	return wrap(style, fmt.Sprintf(format, a...))
}

func ProgressBar(filled int, width int) string {
	if !enabled {
		bar := ""
		for range filled {
			bar += "="
		}
		if filled < width {
			bar += ">"
			for range width - filled - 1 {
				bar += " "
			}
		}
		return bar
	}
	bar := accent
	for range filled {
		bar += "="
	}
	if filled < width {
		bar += bold + ">" + reset + textDim
		for range width - filled - 1 {
			bar += " "
		}
	}
	return bar + reset
}
