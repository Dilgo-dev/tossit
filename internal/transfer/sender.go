package transfer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Dilgo-dev/tossit/internal/crypto"
	"github.com/Dilgo-dev/tossit/internal/progress"
	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/coder/websocket"
)

type SendOptions struct {
	RelayURL string
	Paths    []string
	Stream   bool
}

func Send(ctx context.Context, opts SendOptions) error {
	if len(opts.Paths) == 0 {
		return fmt.Errorf("no files specified")
	}

	info, err := os.Stat(opts.Paths[0])
	if err != nil {
		return err
	}

	isMulti := len(opts.Paths) > 1 || info.IsDir()

	var name string
	var size int64
	if isMulti {
		name = "archive.tar"
		size = 0
	} else {
		name = info.Name()
		size = info.Size()
	}

	conn, _, err := websocket.Dial(ctx, opts.RelayURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}
	conn.SetReadLimit(10 * 1024 * 1024)
	pc := NewPeerConn(ctx, conn)
	defer pc.Close()

	code := protocol.GenerateCode()
	regPayload := append([]byte{0x00}, []byte(code)...)
	if opts.Stream {
		regPayload[0] = 0x01
	}
	if err := pc.SendRaw(protocol.Message{Type: protocol.MsgRegister, Payload: regPayload}); err != nil {
		return err
	}

	httpURL := strings.Replace(opts.RelayURL, "wss://", "https://", 1)
	httpURL = strings.Replace(httpURL, "ws://", "http://", 1)
	httpURL = strings.TrimSuffix(httpURL, "/ws")

	fmt.Println("Code:", code)
	fmt.Printf("On another machine, run: tossit receive %s\n", code)
	fmt.Printf("Or open in browser: %s/d/%s\n", httpURL, code)

	var key []byte

	if opts.Stream {
		fmt.Println("Waiting for receiver...")

		msg, err := pc.RecvRaw()
		if err != nil {
			return err
		}
		if msg.Type == protocol.MsgError {
			return fmt.Errorf("relay: %s", msg.Payload)
		}

		switch msg.Type {
		case protocol.MsgReady:
			fmt.Println("Receiver connected. Establishing secure channel...")
			key, err = crypto.SenderKeyExchange(pc.SendPeer, func() ([]byte, error) {
				return pc.RecvPeer()
			}, code)
			if err != nil {
				return fmt.Errorf("key exchange failed: %w", err)
			}
		case protocol.MsgBrowserJoin:
			fmt.Println("Browser receiver connected. Establishing secure channel...")
			key = crypto.DeriveKeyFromCode(code)
		default:
			return fmt.Errorf("unexpected message from relay: %d", msg.Type)
		}
	} else {
		fmt.Println("Uploading...")
		key = crypto.DeriveKeyFromCode(code)
	}

	meta := protocol.Metadata{
		Name:      name,
		Size:      size,
		Mode:      info.Mode(),
		IsDir:     isMulti,
		ChunkSize: crypto.ChunkSize,
	}
	if isMulti {
		meta.FileCount = len(opts.Paths)
	}

	metaBytes, err := protocol.EncodeMetadata(meta)
	if err != nil {
		return err
	}
	if err := pc.SendPeer(metaBytes); err != nil {
		return err
	}

	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		return err
	}

	if isMulti {
		if err := sendArchive(ctx, pc, enc, opts.Paths); err != nil {
			return err
		}
	} else {
		if err := sendFile(ctx, pc, enc, opts.Paths[0], size); err != nil {
			return err
		}
	}

	if !opts.Stream {
		msg, err := pc.RecvRaw()
		if err != nil {
			return err
		}
		if msg.Type == protocol.MsgStored {
			fmt.Println("Upload complete! File available for download.")
		} else if msg.Type == protocol.MsgError {
			return fmt.Errorf("relay: %s", msg.Payload)
		}
	}

	return nil
}

func sendFile(ctx context.Context, pc *PeerConn, enc *crypto.Encryptor, path string, size int64) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := progress.New(size)
	hasher := sha256.New()
	buf := make([]byte, crypto.ChunkSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := f.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			hasher.Write(chunk)

			seq, ciphertext, encErr := enc.EncryptChunk(chunk)
			if encErr != nil {
				return encErr
			}

			encoded := protocol.EncodeChunk(seq, ciphertext)
			if err := pc.SendPeer(encoded); err != nil {
				return err
			}

			bar.Add(int64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	bar.Done()

	done := protocol.EncodeDone(hasher.Sum(nil))
	return pc.SendPeer(done)
}
