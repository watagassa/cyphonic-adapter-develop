package layers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// padByPkcs7 returns a byte slice with PKCS#7 padding applied.
// The padding size is adjusted to be a multiple of aes.BlockSize.
func padByPkcs7(data []byte) []byte {
	padSize := aes.BlockSize
	if len(data)%aes.BlockSize != 0 {
		padSize = aes.BlockSize - (len(data))%aes.BlockSize
	}

	pad := bytes.Repeat([]byte{byte(padSize)}, padSize)
	return append(data, pad...)
}

// unPadByPkcs7 removes PKCS#7 padding and returns original data.
// The padding size is stored in the last byte of data.
func unPadByPkcs7(data []byte) ([]byte, error) {
	if len(data) == 0 {
		err := errors.New("data slice is empty")
		return nil, err
	}

	padSize := int(data[len(data)-1])

	if padSize >= len(data) {
		return nil, fmt.Errorf("invalid padSize: %d", padSize)
	}

	return data[:len(data)-padSize], nil
}

// EncryptPacket encrypts the packet.
func EncryptPacket(buf, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new cipher block: %w", err)
	}

	paddBuf := padByPkcs7(buf)
	cipherBuf := make([]byte, aes.BlockSize+len(paddBuf))
	iv := cipherBuf[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("failed to generate Initialization Vector: %w", err)
	}

	encryptStream := cipher.NewCBCEncrypter(block, iv)
	encryptStream.CryptBlocks(cipherBuf[aes.BlockSize:], paddBuf)

	return cipherBuf, nil
}

// DecryptPacket decrypts the packet.
func DecryptPacket(buf, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new cipher block: %w", err)
	}

	decryptedText := make([]byte, len(buf[aes.BlockSize:]))
	decryptStream := cipher.NewCBCDecrypter(block, buf[:aes.BlockSize])
	decryptStream.CryptBlocks(decryptedText, buf[aes.BlockSize:])

	origin, err := unPadByPkcs7(decryptedText)
	if err != nil {
		return nil, fmt.Errorf("failed to remove PKCS#7 padding: %w", err)
	}

	return origin, nil
}
