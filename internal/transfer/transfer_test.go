package transfer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dilgo-dev/tossit/internal/relay"
)

func TestSendReceiveFile(t *testing.T) {
	r := relay.New(relay.Config{StorageDir: t.TempDir()})
	srv := &http.Server{Addr: ":0", ReadHeaderTimeout: 10 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", r.HandleConn)
	srv.Handler = mux

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	relayURL := fmt.Sprintf("ws://%s/ws", ln.Addr().String())

	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.bin")
	data := make([]byte, 128*1024) // 128KB
	for i := range data {
		data[i] = byte(i % 256)
	}
	if err := os.WriteFile(testFile, data, 0o600); err != nil {
		t.Fatal(err)
	}

	expectedHash := sha256.Sum256(data)

	// We need to capture the code from Send, so we'll use protocol.GenerateCode
	// and override... Actually, let's just run send and receive in goroutines
	// and capture stdout to get the code.

	// Simpler: use the transfer functions directly but we need the code.
	// Send generates the code internally. Let's just test via goroutines.

	recvDir := filepath.Join(tmpDir, "recv")
	if err := os.MkdirAll(recvDir, 0o750); err != nil {
		t.Fatal(err)
	}

	sendDone := make(chan error, 1)
	codeCh := make(chan string, 1)

	// Capture stdout to get the code
	origStdout := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		err := Send(ctx, SendOptions{
			RelayURL: relayURL,
			Paths:    []string{testFile},
		})
		_ = pw.Close()
		sendDone <- err
	}()

	// Read code from stdout
	go func() {
		buf := make([]byte, 4096)
		n, _ := pr.Read(buf)
		output := string(buf[:n])
		// Parse "Code: xxx\n"
		for _, line := range splitLines(output) {
			if len(line) > 6 && line[:6] == "Code: " {
				codeCh <- line[6:]
				return
			}
		}
	}()

	select {
	case code := <-codeCh:
		os.Stdout = origStdout
		t.Logf("Got code: %s", code)

		err := Receive(ctx, ReceiveOptions{
			RelayURL:  relayURL,
			Code:      code,
			OutputDir: recvDir,
		})
		if err != nil {
			t.Fatalf("Receive failed: %v", err)
		}

		// Verify
		received, err := os.ReadFile(filepath.Clean(filepath.Join(recvDir, "test.bin")))
		if err != nil {
			t.Fatal(err)
		}
		actualHash := sha256.Sum256(received)
		if expectedHash != actualHash {
			t.Fatal("hash mismatch")
		}
		t.Log("File transferred and verified successfully")

	case <-time.After(5 * time.Second):
		os.Stdout = origStdout
		t.Fatal("timeout waiting for code")
	}

	if err := <-sendDone; err != nil {
		t.Logf("Send returned: %v", err)
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
