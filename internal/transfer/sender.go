package transfer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"time"

	"github.com/Dilgo-dev/tossit/internal/color"
	"github.com/Dilgo-dev/tossit/internal/crypto"
	"github.com/Dilgo-dev/tossit/internal/history"
	"github.com/Dilgo-dev/tossit/internal/p2p"
	"github.com/Dilgo-dev/tossit/internal/progress"
	"github.com/Dilgo-dev/tossit/internal/protocol"
	"github.com/Dilgo-dev/tossit/internal/qr"
	"github.com/coder/websocket"
)

type SendOptions struct {
	RelayURL   string
	RelayToken string
	Paths      []string
	Stream     bool
	Stdin      io.Reader
	Password   string
	Expires    time.Duration
	Direct     bool
	StunServer string
	Multi      int
}

func Send(ctx context.Context, opts SendOptions) error {
	isStdin := opts.Stdin != nil
	if !isStdin && len(opts.Paths) == 0 {
		return fmt.Errorf("no files specified")
	}

	var name string
	var size int64
	var fileMode os.FileMode
	var isMulti bool

	if isStdin {
		name = "stdin.txt"
		size = 0
		fileMode = 0o644
	} else {
		info, err := os.Stat(opts.Paths[0])
		if err != nil {
			return err
		}
		isMulti = len(opts.Paths) > 1 || info.IsDir()
		fileMode = info.Mode()
		if isMulti {
			name = "archive.tar"
			size = 0
		} else {
			name = info.Name()
			size = info.Size()
		}
	}

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

	code := protocol.GenerateCode()
	regPayload := append([]byte{0x00}, []byte(code)...)
	if opts.Stream {
		regPayload[0] = 0x01
	}
	if opts.Expires > 0 || opts.Multi > 0 {
		regPayload = append(regPayload, 0x00)
		if opts.Expires > 0 {
			regPayload = append(regPayload, []byte(fmt.Sprintf("%d", int(opts.Expires.Seconds())))...)
		}
	}
	if opts.Multi > 0 {
		regPayload = append(regPayload, 0x00)
		regPayload = append(regPayload, []byte(fmt.Sprintf("%d", opts.Multi))...)
	}
	if err := pc.SendRaw(protocol.Message{Type: protocol.MsgRegister, Payload: regPayload}); err != nil {
		return err
	}

	httpURL := strings.Replace(opts.RelayURL, "wss://", "https://", 1)
	httpURL = strings.Replace(httpURL, "ws://", "http://", 1)
	httpURL = strings.TrimSuffix(httpURL, "/ws")

	browserURL := fmt.Sprintf("%s/d/%s", httpURL, code)
	if opts.RelayToken != "" {
		browserURL += "?token=" + opts.RelayToken
	}

	fmt.Printf("%s %s\n", color.Dim("Code:"), color.BoldCyan(code))
	if opts.Password != "" {
		fmt.Printf("%s tossit receive --password <pw> %s\n", color.Dim("On another machine, run:"), color.BoldCyan(code))
		fmt.Printf("%s %s\n", color.Dim("Password protected:"), color.Yellow("receiver must use the same --password"))
	} else {
		fmt.Printf("%s tossit receive %s\n", color.Dim("On another machine, run:"), color.BoldCyan(code))
	}
	if opts.Expires > 0 {
		fmt.Printf("%s %s\n", color.Dim("Expires in:"), color.Yellow(opts.Expires.String()))
	}
	if opts.Multi > 0 {
		fmt.Printf("%s %s\n", color.Dim("Max downloads:"), color.Yellow(fmt.Sprintf("%d", opts.Multi)))
	}
	fmt.Printf("%s %s\n", color.Dim("Or open in browser:"), color.Cyan(browserURL))
	qr.Print(browserURL)

	var key []byte
	var t Transport = pc

	if opts.Stream {
		fmt.Println(color.Dim("Waiting for receiver..."))

		msg, err := pc.RecvRaw()
		if err != nil {
			return err
		}
		if msg.Type == protocol.MsgError {
			return fmt.Errorf("relay: %s", msg.Payload)
		}

		switch msg.Type {
		case protocol.MsgReady:
			fmt.Println(color.Green("Receiver connected."), color.Dim("Establishing secure channel..."))

			if opts.Direct {
				if udpT, p2pErr := negotiateP2PSender(ctx, pc, opts.StunServer); p2pErr == nil {
					fmt.Println(color.Green("Direct P2P connection established."))
					t = udpT
				} else {
					fmt.Println(color.Yellow("P2P failed, falling back to relay."))
				}
			}

			key, err = crypto.SenderKeyExchange(t.SendPeer, func() ([]byte, error) {
				return t.RecvPeer()
			}, code)
			if err != nil {
				return fmt.Errorf("key exchange failed: %w", err)
			}
		case protocol.MsgBrowserJoin:
			fmt.Println(color.Green("Browser receiver connected."), color.Dim("Establishing secure channel..."))
			key = crypto.DeriveKeyFromCode(code, opts.Password)
		default:
			return fmt.Errorf("unexpected message from relay: %d", msg.Type)
		}
	} else {
		fmt.Println(color.Dim("Uploading..."))
		key = crypto.DeriveKeyFromCode(code, opts.Password)
	}

	meta := protocol.Metadata{
		Name:      name,
		Size:      size,
		Mode:      fileMode,
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
	if err := t.SendPeer(metaBytes); err != nil {
		return err
	}

	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		return err
	}

	switch {
	case isStdin:
		if err := sendReader(ctx, t, enc, opts.Stdin); err != nil {
			return err
		}
	case isMulti:
		if err := sendArchive(ctx, t, enc, opts.Paths); err != nil {
			return err
		}
	default:
		if err := sendFile(ctx, t, enc, opts.Paths[0], size); err != nil {
			return err
		}
	}

	if !opts.Stream {
		msg, err := pc.RecvRaw()
		if err != nil {
			return err
		}
		switch msg.Type {
		case protocol.MsgStored:
			fmt.Println(color.Green("Upload complete!"), "File available for download.")
		case protocol.MsgError:
			return fmt.Errorf("relay: %s", msg.Payload)
		}
	}

	history.Add(history.Entry{
		Direction: history.Sent,
		Name:      name,
		Size:      size,
		Code:      code,
		Time:      time.Now(),
	})

	return nil
}

func sendFile(ctx context.Context, t Transport, enc *crypto.Encryptor, path string, size int64) error {
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
			if err := t.SendPeer(encoded); err != nil {
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
	return t.SendPeer(done)
}

func sendReader(ctx context.Context, t Transport, enc *crypto.Encryptor, r io.Reader) error {
	counter := progress.NewCounter()
	hasher := sha256.New()
	buf := make([]byte, crypto.ChunkSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := r.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			hasher.Write(chunk)

			seq, ciphertext, encErr := enc.EncryptChunk(chunk)
			if encErr != nil {
				return encErr
			}

			encoded := protocol.EncodeChunk(seq, ciphertext)
			if sendErr := t.SendPeer(encoded); sendErr != nil {
				return sendErr
			}

			counter.Add(int64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	counter.Done()

	done := protocol.EncodeDone(hasher.Sum(nil))
	return t.SendPeer(done)
}

func negotiateP2PSender(ctx context.Context, pc *PeerConn, stunServer string) (Transport, error) {
	udpConn, candidates, err := p2p.GatherCandidates(stunServer)
	if err != nil {
		return nil, err
	}

	candidateData, err := protocol.EncodeCandidates(candidates)
	if err != nil {
		_ = udpConn.Close()
		return nil, err
	}
	offer := append([]byte{protocol.PeerP2POffer}, candidateData...)
	if err := pc.SendPeer(offer); err != nil {
		_ = udpConn.Close()
		return nil, err
	}

	payload, err := pc.RecvPeer()
	if err != nil {
		_ = udpConn.Close()
		return nil, err
	}
	if len(payload) == 0 {
		_ = udpConn.Close()
		return nil, fmt.Errorf("empty P2P response")
	}
	if payload[0] == protocol.PeerP2PReject {
		_ = udpConn.Close()
		return nil, fmt.Errorf("receiver rejected P2P")
	}
	if payload[0] != protocol.PeerP2PAccept {
		_ = udpConn.Close()
		return nil, fmt.Errorf("unexpected P2P response: 0x%02x", payload[0])
	}

	remoteCandidates, err := protocol.DecodeCandidates(payload[1:])
	if err != nil {
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
