// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// HolePunching is HolePunching of a CYPHONIC packet.
type HolePunching struct {
	BaseHeader BaseHeader
	HMAC       HMAC
}

// GenerateHolePunching generates HolePunching packet.
func (hp *HolePunching) GenerateHolePunching(pathID ID) {

	if err := SerializeBaseHeader(&hp.BaseHeader); err != nil {
		logger.Error(fmt.Errorf("failed to serialize BaseHeader: %w", err))
	}

	SerializeType(&hp.BaseHeader, TypeClassHolePunching)

	hp.BaseHeader.ID = pathID

	// FIXME: HMAC is fixed.
	hp.HMAC = []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}

	// FIXME: calculate packet length.
	hp.BaseHeader.MessageLength = 48

	logger.Debug("Generate Hole Punching", hp)
}

// Marshal returns binary data.
func (hp *HolePunching) Marshal() ([]byte, error) {
	buf := make([]byte, 0, hp.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.TransactionID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.Flag); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.Count); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.MessageLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.NextOpt); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.BaseHeader.ID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, hp.HMAC); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// UnmarshalHolePunching unmarshals a structure from binary.
func UnmarshalHolePunching(b []byte) (HolePunching, error) {
	hp := HolePunching{
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

	return hp, nil
}
