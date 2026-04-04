package transfer

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Dilgo-dev/tossit/internal/color"
	"github.com/Dilgo-dev/tossit/internal/crypto"
	"github.com/Dilgo-dev/tossit/internal/protocol"
)

func sendArchive(ctx context.Context, t Transport, enc *crypto.Encryptor, paths []string) error {
	pr, pw := io.Pipe()

	hasher := sha256.New()
	errCh := make(chan error, 1)

	go func() {
		tw := tar.NewWriter(pw)
		var writeErr error
		for _, p := range paths {
			if writeErr = addToTar(tw, p); writeErr != nil {
				break
			}
		}
		tw.Close()
		pw.CloseWithError(writeErr)
	}()

	go func() {
		buf := make([]byte, crypto.ChunkSize)
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			n, err := pr.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				hasher.Write(chunk)

				seq, ciphertext, encErr := enc.EncryptChunk(chunk)
				if encErr != nil {
					errCh <- encErr
					return
				}
				encoded := protocol.EncodeChunk(seq, ciphertext)
				if sendErr := t.SendPeer(encoded); sendErr != nil {
					errCh <- sendErr
					return
				}
			}
			if err == io.EOF {
				done := protocol.EncodeDone(hasher.Sum(nil))
				errCh <- t.SendPeer(done)
				return
			}
			if err != nil {
				errCh <- err
				return
			}
		}
	}()

	err := <-errCh
	if err != nil {
		return err
	}
	fmt.Println(color.Green("Transfer complete."))
	return nil
}

func receiveArchive(ctx context.Context, t Transport, dec *crypto.Decryptor, outputDir string) error {
	pr, pw := io.Pipe()
	hasher := sha256.New()
	var expectedHash []byte

	errCh := make(chan error, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				pw.CloseWithError(ctx.Err())
				return
			default:
			}

			payload, err := t.RecvPeer()
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			if len(payload) > 0 && payload[0] == protocol.PeerDone {
				h, hashErr := protocol.DecodeDone(payload)
				if hashErr != nil {
					pw.CloseWithError(hashErr)
					return
				}
				expectedHash = h
				pw.Close()
				return
			}

			seq, ciphertext, chunkErr := protocol.DecodeChunk(payload)
			if chunkErr != nil {
				pw.CloseWithError(chunkErr)
				return
			}

			plaintext, decErr := dec.DecryptChunk(seq, ciphertext)
			if decErr != nil {
				pw.CloseWithError(decErr)
				return
			}

			hasher.Write(plaintext)
			if _, writeErr := pw.Write(plaintext); writeErr != nil {
				return
			}
		}
	}()

	go func() {
		errCh <- extractTar(pr, outputDir)
	}()

	err := <-errCh
	if err != nil {
		return err
	}

	if expectedHash != nil {
		actualHash := hasher.Sum(nil)
		if !hashEqual(expectedHash, actualHash) {
			return fmt.Errorf("archive hash mismatch: transfer corrupted")
		}
	}

	fmt.Printf("%s %s\n", color.Green("Extracted to"), color.Bold(outputDir))
	return nil
}

func addToTar(tw *tar.Writer, path string) error {
	return filepath.Walk(path, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(filepath.Dir(path), file)
		if err != nil {
			return err
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

func extractTar(r io.Reader, outputDir string) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target := filepath.Join(outputDir, header.Name)

		if !filepath.IsAbs(target) {
			target = filepath.Join(outputDir, filepath.Clean(header.Name))
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
}
