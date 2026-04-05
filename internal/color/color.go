package color

import (
	"fmt"
	"os"
	"strings"
)

var enabled = func() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	o, _ := os.Stdout.Stat()
	return o.Mode()&os.ModeCharDevice != 0
}()

var trueColor = func() bool {
	ct := os.Getenv("COLORTERM")
	if ct == "truecolor" || ct == "24bit" {
		return true
	}
	// Windows Terminal supports true color
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	if os.Getenv("TERM_PROGRAM") == "Apple_Terminal" {
		return false
	}
	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") {
		tp := os.Getenv("TERM_PROGRAM")
		if tp == "iTerm.app" || tp == "WezTerm" || tp == "ghostty" {
			return true
		}
	}
	return false
}()

var (
	rstSeq = "\033[0m"
	bld = "\033[1m"

	accentSeq    string
	accentDimSeq string
	successSeq   string
	textDimSeq   string
	redSeq       string
)

func init() {
	if trueColor {
		accentSeq = "\033[38;2;163;230;53m"
		accentDimSeq = "\033[38;2;101;163;13m"
		successSeq = "\033[38;2;52;211;153m"
		textDimSeq = "\033[38;2;139;139;150m"
		redSeq = "\033[38;2;255;95;87m"
	} else {
		accentSeq = "\033[38;5;155m"
		accentDimSeq = "\033[38;5;106m"
		successSeq = "\033[38;5;79m"
		textDimSeq = "\033[38;5;246m"
		redSeq = "\033[38;5;203m"
	}
}

func wrap(style, s string) string {
	if !enabled {
		return s
	}
	return style + s + rstSeq
}

func Bold(s string) string      { return wrap(bld, s) }
func Dim(s string) string       { return wrap(textDimSeq, s) }
func Red(s string) string       { return wrap(redSeq, s) }
func Green(s string) string     { return wrap(successSeq, s) }
func Yellow(s string) string    { return wrap(accentDimSeq, s) }
func Cyan(s string) string      { return wrap(accentSeq, s) }
func BoldCyan(s string) string  { return wrap(bld+accentSeq, s) }
func BoldRed(s string) string   { return wrap(bld+redSeq, s) }
func Accent(s string) string    { return wrap(accentSeq, s) }
func AccentDim(s string) string { return wrap(accentDimSeq, s) }

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
	bar := accentSeq
	for range filled {
		bar += "="
	}
	if filled < width {
		bar += bld + ">" + rstSeq + textDimSeq
		for range width - filled - 1 {
			bar += " "
		}
	}
	return bar + rstSeq
}
