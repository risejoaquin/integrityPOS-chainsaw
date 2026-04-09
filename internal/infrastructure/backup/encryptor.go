package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

// Encryptor handles AES encryption/decryption
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given key
func NewEncryptor(keyString string) (*Encryptor, error) {
	if len(keyString) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}

	return &Encryptor{
		key: []byte(keyString),
	}, nil
}

// EncryptFile encrypts a file using AES-256
func (e *Encryptor) EncryptFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Create cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return err
	}

	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return err
	}

	// Write IV to destination
	if _, err := dst.Write(iv); err != nil {
		return err
	}

	// Create stream cipher
	stream := cipher.NewCFBEncrypter(block, iv)

	// Encrypt and write
	writer := cipher.StreamWriter{S: stream, W: dst}
	_, err = io.Copy(writer, src)
	return err
}

// DecryptFile decrypts a file using AES-256
func (e *Encryptor) DecryptFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Create cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return err
	}

	// Read IV from source
	iv := make([]byte, aes.BlockSize)
	if _, err := src.Read(iv); err != nil {
		return err
	}

	// Create stream cipher
	stream := cipher.NewCFBDecrypter(block, iv)

	// Decrypt and write
	reader := cipher.StreamReader{S: stream, R: src}
	_, err = io.Copy(dst, reader)
	return err
}

// EncryptReader returns an encrypted reader
func (e *Encryptor) EncryptReader(reader io.Reader) (io.Reader, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)

	// Prepend IV to encrypted stream
	ivReader := &prefixReader{prefix: iv, reader: reader}
	return cipher.StreamReader{S: stream, R: ivReader}, nil
}

// DecryptReader returns a decrypted reader
func (e *Encryptor) DecryptReader(reader io.Reader) (io.Reader, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	// Read IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBDecrypter(block, iv)
	return cipher.StreamReader{S: stream, R: reader}, nil
}

// prefixReader prepends a prefix to a reader
type prefixReader struct {
	prefix []byte
	reader io.Reader
	pos    int
}

func (pr *prefixReader) Read(p []byte) (n int, err error) {
	// First return prefix
	if pr.pos < len(pr.prefix) {
		n = copy(p, pr.prefix[pr.pos:])
		pr.pos += n
		return n, nil
	}

	// Then return from underlying reader
	return pr.reader.Read(p)
}