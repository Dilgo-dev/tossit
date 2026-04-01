package progress

import (
	"fmt"
	"strings"
	"time"
)

type Bar struct {
	total   int64
	current int64
	start   time.Time
	width   int
}

func New(total int64) *Bar {
	return &Bar{total: total, start: time.Now(), width: 40}
}

func (b *Bar) Add(n int64) {
	b.current += n
	b.render()
}

func (b *Bar) Done() {
	b.current = b.total
	b.render()
	fmt.Println()
}

func (b *Bar) render() {
	elapsed := time.Since(b.start).Seconds()
	if elapsed < 0.1 {
		elapsed = 0.1
	}

	pct := float64(b.current) / float64(b.total)
	if pct > 1 {
		pct = 1
	}

	filled := int(pct * float64(b.width))
	bar := strings.Repeat("=", filled)
	if filled < b.width {
		bar += ">"
		bar += strings.Repeat(" ", b.width-filled-1)
	}

	speed := float64(b.current) / elapsed
	eta := ""
	if speed > 0 && b.current < b.total {
		remaining := float64(b.total-b.current) / speed
		eta = formatDuration(remaining)
	}

	fmt.Printf("\r[%s] %3.0f%% %s %s  ",
		bar,
		pct*100,
		formatSpeed(speed),
		eta,
	)
}

func formatSpeed(bytesPerSec float64) string {
	switch {
	case bytesPerSec >= 1<<30:
		return fmt.Sprintf("%.1f GB/s", bytesPerSec/float64(1<<30))
	case bytesPerSec >= 1<<20:
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/float64(1<<20))
	case bytesPerSec >= 1<<10:
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/float64(1<<10))
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
}

func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("ETA %ds", int(seconds))
	}
	m := int(seconds) / 60
	s := int(seconds) % 60
	return fmt.Sprintf("ETA %d:%02d", m, s)
}

func FormatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
