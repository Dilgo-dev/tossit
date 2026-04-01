package transfer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dilgo-dev/tossit/internal/crypto"
	"github.com/Dilgo-dev/tossit/internal/progress"
	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/coder/websocket"
)

type ReceiveOptions struct {
	RelayURL  string
	Code      string
	OutputDir string
}

func Receive(ctx context.Context, opts ReceiveOptions) error {
	conn, _, err := websocket.Dial(ctx, opts.RelayURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}
	conn.SetReadLimit(10 * 1024 * 1024)
	pc := NewPeerConn(ctx, conn)
	defer pc.Close()

	if err := pc.SendRaw(protocol.Message{Type: protocol.MsgJoin, Payload: []byte(opts.Code)}); err != nil {
		return err
	}

	msg, err := pc.RecvRaw()
	if err != nil {
		return err
	}
	if msg.Type == protocol.MsgError {
		return fmt.Errorf("relay: %s", msg.Payload)
	}

	var key []byte
	switch msg.Type {
	case protocol.MsgStored:
		fmt.Println("Downloading stored transfer...")
		key = crypto.DeriveKeyFromCode(opts.Code)
	case protocol.MsgData:
		fmt.Println("Establishing secure channel...")
		first := true
		key, err = crypto.ReceiverKeyExchange(pc.SendPeer, func() ([]byte, error) {
			if first {
				first = false
				return msg.Payload, nil
			}
			return pc.RecvPeer()
		}, opts.Code)
		if err != nil {
			return fmt.Errorf("key exchange failed: %w", err)
		}
	default:
		return fmt.Errorf("unexpected message from relay: %d", msg.Type)
	}

	payload, err := pc.RecvPeer()
	if err != nil {
		return err
	}
	meta, err := protocol.DecodeMetadata(payload)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	fmt.Printf("Receiving: %s (%s)\n", meta.Name, progress.FormatSize(meta.Size))

	dec, err := crypto.NewDecryptor(key)
	if err != nil {
		return err
	}

	if meta.IsDir {
		return receiveArchive(ctx, pc, dec, opts.OutputDir)
	}
	return receiveFile(ctx, pc, dec, meta, opts.OutputDir)
}

func receiveFile(ctx context.Context, pc *PeerConn, dec *crypto.Decryptor, meta protocol.Metadata, outputDir string) error {
	outPath := filepath.Join(outputDir, meta.Name)
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, meta.Mode)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := progress.New(meta.Size)
	hasher := sha256.New()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		payload, err := pc.RecvPeer()
		if err != nil {
			return err
		}

		if len(payload) > 0 && payload[0] == protocol.PeerDone {
			expectedHash, hashErr := protocol.DecodeDone(payload)
			if hashErr != nil {
				return hashErr
			}
			actualHash := hasher.Sum(nil)
			if !hashEqual(expectedHash, actualHash) {
				os.Remove(outPath)
				return fmt.Errorf("file hash mismatch: transfer corrupted")
			}
			bar.Done()
			fmt.Printf("Saved to %s\n", outPath)
			return nil
		}

		seq, ciphertext, chunkErr := protocol.DecodeChunk(payload)
		if chunkErr != nil {
			return chunkErr
		}

		plaintext, decErr := dec.DecryptChunk(seq, ciphertext)
		if decErr != nil {
			return decErr
		}

		hasher.Write(plaintext)
		if _, err := f.Write(plaintext); err != nil {
			return err
		}
		bar.Add(int64(len(plaintext)))
	}
}

func hashEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
