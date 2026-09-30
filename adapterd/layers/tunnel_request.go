// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/google/uuid"
)

// TunnelRequest is Tunnel Request of a CYPHONIC packet.
type TunnelRequest struct {
	BaseHeader   BaseHeader
	EndKeyLength uint16
	Padding      uint16
	EndKey       []byte
	HMAC         HMAC
}

// GenerateTunnelRequest generates Tunnle Request packet.
func (treq *TunnelRequest) GenerateTunnelRequest(pathID ID, tempKey []byte, process ProcessCodeClass) []byte {
	if err := SerializeBaseHeader(&treq.BaseHeader); err != nil {
		logger.Error(fmt.Errorf("failed to serialize BaseHeader: %w", err))
	}
	SerializeType(&treq.BaseHeader, TypeClassTunnelRequest)
	treq.BaseHeader.ID = pathID
	endKey, err := treq.generateEndKey()
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate end key: %w", err))
	}

	if process == TunnelRequestToTRS {
		treq.encryptEndKey(tempKey)
	}

	treq.calculateEndKeyLength()

	// FIXME: calculate packet length.
	treq.BaseHeader.MessageLength = 32 + 4 + treq.EndKeyLength + 16

	return endKey
}

// calculateEndKeyLength calculates end key's length to encrypt the contents of the packet.
func (treq *TunnelRequest) calculateEndKeyLength() uint16 {
	len := len(treq.EndKey)
	treq.EndKeyLength = uint16(len)

	return uint16(len)
}

// generateEndKey generates an end key to encrypt the contents of the packet.
func (treq *TunnelRequest) generateEndKey() ([]byte, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	ub, err := u.MarshalBinary()
	if err != nil {
		return nil, err
	}

	treq.EndKey = ub

	return ub, nil
}

// encryptEndKey encrypt the end key.
func (treq *TunnelRequest) encryptEndKey(key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new cipher block: %w", err)
	}

	paddBuf := padByPkcs7(treq.EndKey)
	ek := make([]byte, aes.BlockSize+len(paddBuf))
	iv := ek[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("failed to generate Initialization Vector: %w", err)
	}

	encryptStream := cipher.NewCBCEncrypter(block, iv)
	encryptStream.CryptBlocks(ek[aes.BlockSize:], paddBuf)

	treq.EndKey = ek

	return ek, nil
}

// DecryptEndKey decrypt the end key.
func (treq *TunnelRequest) DecryptEndKey(key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new cipher block: %w", err)
	}

	dk := make([]byte, len(treq.EndKey[aes.BlockSize:]))
	decryptStream := cipher.NewCBCDecrypter(block, treq.EndKey[:aes.BlockSize])
	decryptStream.CryptBlocks(dk, treq.EndKey[aes.BlockSize:])

	return dk, nil
}

// Marshal returns binary data.
func (treq *TunnelRequest) Marshal(key []byte) ([]byte, error) {
	planeBuf := make([]byte, 0, treq.BaseHeader.MessageLength-BaseHeaderLen-HMACLen)
	planeBuffer := bytes.NewBuffer(planeBuf)

	if err := binary.Write(planeBuffer, binary.BigEndian, treq.EndKeyLength); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, treq.Padding); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, treq.EndKey); err != nil {
		return nil, err
	}

	chiperBuf, err := EncryptPacket(planeBuffer.Bytes(), key)
	if err != nil {
		return nil, err
	}

	treq.BaseHeader.MessageLength = BaseHeaderLen + uint16(len(chiperBuf)) + HMACLen

	buf := make([]byte, 0, treq.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.TransactionID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.Flag); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.Count); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.MessageLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.NextOpt); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, treq.BaseHeader.ID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, chiperBuf); err != nil {
		return nil, err
	}

	treq.HMAC = GenerateHMAC(buffer.Bytes(), GenerateSalt(&treq.BaseHeader))

	if err := binary.Write(buffer, binary.BigEndian, treq.HMAC); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// UnmarshalTunnelRequest unmarshals a structure from binary.
func UnmarshalTunnelRequest(b, key []byte) (TunnelRequest, error) {
	encryptedLen := binary.BigEndian.Uint16(b[12:14]) - HMACLen
	pb, err := DecryptPacket(b[32:encryptedLen], key)
	if err != nil {
		return TunnelRequest{}, fmt.Errorf("failed to decrypt Tunnel Request: %w", err)
	}

	salt := InitSalt(b)

	if ok := DecodeHMAC(b[encryptedLen:encryptedLen+HMACLen], b[0:encryptedLen], salt); !ok {
		err := errors.New("decode HMAC error")
		return TunnelRequest{}, err
	}
	endKeyLength := 4 + binary.BigEndian.Uint16(pb[0:2])
	tr := TunnelRequest{
		BaseHeader: BaseHeader{
			TransactionID:  binary.BigEndian.Uint32(b[0:4]),
			Version:        b[4],
			Flag:           b[5],
			Type:           b[6],
			Count:          b[7],
			SequenceNumber: binary.BigEndian.Uint32(b[8:12]),
			MessageLength:  binary.BigEndian.Uint16(b[12:14]),
			NextOpt:        b[14],
			Reserved:       b[15],
			ID:             b[16:32],
		},
		EndKeyLength: binary.BigEndian.Uint16(pb[0:2]),
		Padding:      binary.BigEndian.Uint16(pb[2:4]),
		EndKey:       pb[4:endKeyLength],
		HMAC:         b[encryptedLen : encryptedLen+HMACLen],
	}

	return tr, nil
}
