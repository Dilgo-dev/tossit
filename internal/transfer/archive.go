package transfer

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
		_ = tw.Close()
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
				_ = pw.Close()
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
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	absBase := filepath.Dir(absPath)

	var files []string
	err = filepath.WalkDir(absPath, func(file string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		files = append(files, file)
		return nil
	})
	if err != nil {
		return err
	}

	for _, file := range files {
		fi, err := os.Lstat(file)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(absBase, file)
		if err != nil {
			return err
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if fi.IsDir() {
			continue
		}

		if err := copyFileToTar(tw, file); err != nil {
			return err
		}
	}
	return nil
}

func copyFileToTar(tw *tar.Writer, path string) error {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(tw, f)
	return err
}

const maxExtractSize = 10 * 1024 * 1024 * 1024

func extractTar(r io.Reader, outputDir string) error {
	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		clean := filepath.Clean(header.Name)
		target := filepath.Join(absOutput, clean)
		if !strings.HasPrefix(target, absOutput+string(os.PathSeparator)) && target != absOutput {
			return fmt.Errorf("tar entry %q escapes output directory", header.Name)
		}

		mode := header.Mode & 0o777

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(mode)&0o750); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
				return err
			}
			if err := extractFile(target, tr, os.FileMode(mode)); err != nil {
				return err
			}
		}
	}
}

func extractFile(path string, r io.Reader, mode os.FileMode) error {
	f, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, io.LimitReader(r, maxExtractSize))
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}
