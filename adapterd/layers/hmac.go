// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"crypto/hmac"
	"crypto/md5" // G501: Blacklisted import `crypto/md5`: weak cryptographic primitive
	"encoding/binary"
	"fmt"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// HMACLen lengths (bytes).
const HMACLen = 16

// HMAC is Hash-based Message Authentication Code of a CYPHONIC packet.
// length of the byte slice: a 16-byte slice.
type HMAC []byte

// HMACCode is Hash-based Message Authentication Code of a CYPHONIC packet.
type HMACCode struct {
	HMAC HMAC // Hash-based Message Authentication Code.
}

// InitSalt is to string with 8 bytes of Transaction ID & Sequence Number.
func InitSalt(b []byte) []byte {
	tid := make([]byte, 0)
	sn := make([]byte, 0)

	tid = append(tid, b[0:4]...)
	sn = append(sn, b[8:12]...)
	salt := append(tid, sn...)

	return salt
}

// GenerateSalt is to string with 8 bytes of Transaction ID & Sequence Number.
func GenerateSalt(b *BaseHeader) []byte {
	tid := make([]byte, 4)
	sn := make([]byte, 4)

	binary.BigEndian.PutUint32(tid, b.TransactionID)
	binary.BigEndian.PutUint32(sn, b.SequenceNumber)
	salt := append(tid, sn...)

	return salt
}

// GenerateHMAC is to generate HMAC.
func GenerateHMAC(message, salt []byte) HMAC {
	mac := hmac.New(md5.New, salt)
	if _, err := mac.Write(message); err != nil {
		logger.Error(fmt.Errorf("failed to generate HMAC: %w", err))
	}

	expectedMAC := mac.Sum(nil)

	return expectedMAC
}

// DecodeHMAC is to decode HMAC to determine integrity.
func DecodeHMAC(hm, message, salt []byte) bool {
	mac := hmac.New(md5.New, salt)
	if _, err := mac.Write(message); err != nil {
		logger.Error(fmt.Errorf("failed to decode HMAC: %w", err))
	}

	expectedMAC := mac.Sum(nil)

	return hmac.Equal(hm, expectedMAC)
}
