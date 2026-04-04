package transfer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/Dilgo-dev/tossit/internal/color"
	"github.com/Dilgo-dev/tossit/internal/crypto"
	"github.com/Dilgo-dev/tossit/internal/history"
	"github.com/Dilgo-dev/tossit/internal/p2p"
	"github.com/Dilgo-dev/tossit/internal/progress"
	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/coder/websocket"
)

type ReceiveOptions struct {
	RelayURL   string
	RelayToken string
	Code       string
	OutputDir  string
	Password   string
	Direct     bool
	StunServer string
}

func Receive(ctx context.Context, opts ReceiveOptions) error {
	dialURL := opts.RelayURL
	if opts.RelayToken != "" {
		sep := "?"
		if strings.Contains(dialURL, "?") {
			sep = "&"
		}
		dialURL += sep + "token=" + opts.RelayToken
	}

	conn, _, err := websocket.Dial(ctx, dialURL, nil)
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

	if msg.Type == protocol.MsgWaiting {
		fmt.Println(color.Dim("Waiting for sender approval..."))
		msg, err = pc.RecvRaw()
		if err != nil {
			return err
		}
		if msg.Type == protocol.MsgError {
			return fmt.Errorf("relay: %s", msg.Payload)
		}
	}

	var key []byte
	var t Transport = pc

	switch msg.Type {
	case protocol.MsgStored:
		fmt.Println(color.Dim("Downloading stored transfer..."))
		key = crypto.DeriveKeyFromCode(opts.Code, opts.Password)
	case protocol.MsgData:
		fmt.Println(color.Dim("Establishing secure channel..."))

		if len(msg.Payload) > 0 && msg.Payload[0] == protocol.PeerP2POffer {
			if udpT, p2pErr := negotiateP2PReceiver(ctx, pc, msg.Payload[1:], opts.StunServer); p2pErr == nil {
				fmt.Println(color.Green("Direct P2P connection established."))
				t = udpT
			} else {
				fmt.Println(color.Yellow("P2P failed, falling back to relay."))
				reject := []byte{protocol.PeerP2PReject}
				_ = pc.SendPeer(reject)
			}
			key, err = crypto.ReceiverKeyExchange(t.SendPeer, func() ([]byte, error) {
				return t.RecvPeer()
			}, opts.Code)
			if err != nil {
				return fmt.Errorf("key exchange failed: %w", err)
			}
		} else {
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
		}
	default:
		return fmt.Errorf("unexpected message from relay: %d", msg.Type)
	}

	payload, err := t.RecvPeer()
	if err != nil {
		return err
	}
	meta, err := protocol.DecodeMetadata(payload)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	fmt.Printf("%s %s %s\n", color.Dim("Receiving:"), color.Bold(meta.Name), color.Dim("("+progress.FormatSize(meta.Size)+")"))

	dec, err := crypto.NewDecryptor(key)
	if err != nil {
		return err
	}

	if meta.IsDir {
		if err := receiveArchive(ctx, t, dec, opts.OutputDir); err != nil {
			return err
		}
	} else {
		if err := receiveFile(ctx, t, dec, meta, opts.OutputDir); err != nil {
			return err
		}
	}

	_ = pc.SendRaw(protocol.Message{Type: protocol.MsgDeleteOK})

	history.Add(history.Entry{
		Direction: history.Received,
		Name:      meta.Name,
		Size:      meta.Size,
		Code:      opts.Code,
		Time:      time.Now(),
	})

	return nil
}

func receiveFile(ctx context.Context, t Transport, dec *crypto.Decryptor, meta protocol.Metadata, outputDir string) error {
	outPath := filepath.Clean(filepath.Join(outputDir, filepath.Base(meta.Name)))
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, meta.Mode)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	bar := progress.New(meta.Size)
	hasher := sha256.New()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		payload, err := t.RecvPeer()
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
				_ = os.Remove(outPath)
				return fmt.Errorf("file hash mismatch: transfer corrupted")
			}
			bar.Done()
			fmt.Printf("%s %s\n", color.Green("Saved to"), color.Bold(outPath))
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

func negotiateP2PReceiver(ctx context.Context, pc *PeerConn, offerPayload []byte, stunServer string) (Transport, error) {
	remoteCandidates, err := protocol.DecodeCandidates(offerPayload)
	if err != nil {
		return nil, err
	}

	udpConn, candidates, err := p2p.GatherCandidates(stunServer)
	if err != nil {
		return nil, err
	}

	candidateData, err := protocol.EncodeCandidates(candidates)
	if err != nil {
		_ = udpConn.Close()
		return nil, err
	}
	accept := append([]byte{protocol.PeerP2PAccept}, candidateData...)
	if err := pc.SendPeer(accept); err != nil {
		_ = udpConn.Close()
		return nil, err
	}

	remoteAddr, err := p2p.HolePunch(ctx, udpConn, remoteCandidates, 5*time.Second)
	if err != nil {
		_ = udpConn.Close()
		return nil, err
	}

	return p2p.NewUDPConn(ctx, udpConn, remoteAddr), nil
}
