// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// KeepAlive is keep alive of a CYPHONIC packet.
type KeepAlive struct {
	BaseHeader BaseHeader
	HMAC       HMAC
}

// GenerateKeepAlive generates keep alive packet.
func GenerateKeepAlive(id ID) KeepAlive {
	kp := KeepAlive{}
	if err := SerializeBaseHeader(&kp.BaseHeader); err != nil {
		logger.Error(fmt.Errorf("failed to serialize BaseHeader: %w", err))
	}
	SerializeType(&kp.BaseHeader, TypeClassKeepAlive)

	// FIXME: calculate packet length.
	kp.BaseHeader.MessageLength = 48
	kp.BaseHeader.ID = id

	// FIXME: HMAC is fixed.
	kp.HMAC = []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}

	return kp
}

// Marshal returns binary data.
func (keepAlive *KeepAlive) Marshal() ([]byte, error) {
	buf := make([]byte, 0, keepAlive.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.TransactionID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.Flag); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.Count); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.MessageLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.NextOpt); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.BaseHeader.ID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, keepAlive.HMAC); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
