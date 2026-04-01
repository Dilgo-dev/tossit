package transfer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/Dilgo-dev/tossit/internal/crypto"
	"github.com/Dilgo-dev/tossit/internal/progress"
	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/coder/websocket"
)

type SendOptions struct {
	RelayURL string
	Paths    []string
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
	if err := pc.SendRaw(protocol.Message{Type: protocol.MsgRegister, Payload: []byte(code)}); err != nil {
		return err
	}

	fmt.Println("Code:", code)
	fmt.Printf("On another machine, run: tossit receive %s\n", code)
	fmt.Println("Waiting for receiver...")

	msg, err := pc.RecvRaw()
	if err != nil {
		return err
	}
	if msg.Type == protocol.MsgError {
		return fmt.Errorf("relay: %s", msg.Payload)
	}
	if msg.Type != protocol.MsgReady {
		return fmt.Errorf("unexpected message from relay: %d", msg.Type)
	}

	fmt.Println("Receiver connected. Establishing secure channel...")

	key, err := crypto.SenderKeyExchange(pc.SendPeer, func() ([]byte, error) {
		return pc.RecvPeer()
	}, code)
	if err != nil {
		return fmt.Errorf("key exchange failed: %w", err)
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
		return sendArchive(ctx, pc, enc, opts.Paths)
	}
	return sendFile(ctx, pc, enc, opts.Paths[0], size)
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
	if err := pc.SendPeer(done); err != nil {
		return err
	}

	fmt.Printf("Sent %s (%s)\n", path, progress.FormatSize(size))
	return nil
}
