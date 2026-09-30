// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// TunnelResponse is Tunnel Response of a CYPHONIC packet.
type TunnelResponse struct {
	BaseHeader BaseHeader
	HMAC       HMAC
}

// GenerateTunnelResponse generates Tunnle Response packet.
func (tres *TunnelResponse) GenerateTunnelResponse(pathID ID) {
	if err := SerializeBaseHeader(&tres.BaseHeader); err != nil {
		logger.Error(fmt.Errorf("failed to serialize BaseHeader: %w", err))
	}
	SerializeType(&tres.BaseHeader, TypeClassTunnelResponse)
	tres.BaseHeader.ID = pathID
	tres.BaseHeader.MessageLength = BaseHeaderLen + HMACLen
}

// Marshal returns binary data.
func (tres *TunnelResponse) Marshal() ([]byte, error) {
	buf := make([]byte, 0, tres.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.TransactionID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.Flag); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.Count); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.MessageLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.NextOpt); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, tres.BaseHeader.ID); err != nil {
		return nil, err
	}

	tres.HMAC = GenerateHMAC(buffer.Bytes(), GenerateSalt(&tres.BaseHeader))

	if err := binary.Write(buffer, binary.BigEndian, tres.HMAC); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// UnmarshalTunnelResponse unmarshals a structure from binary.
func UnmarshalTunnelResponse(b []byte) (TunnelResponse, error) {
	encryptedLen := binary.BigEndian.Uint16(b[12:14]) - HMACLen
	salt := InitSalt(b)

	if ok := DecodeHMAC(b[encryptedLen:encryptedLen+HMACLen], b[0:encryptedLen], salt); !ok {
		err := errors.New("decode HMAC error")
		return TunnelResponse{}, err
	}

	tr := TunnelResponse{
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
		HMAC: b[32:48],
	}

	return tr, nil
}
