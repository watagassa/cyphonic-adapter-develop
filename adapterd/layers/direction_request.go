// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/google/uuid"
)

// DirectionRequest is Direction Request of a CYPHONIC packet.
type DirectionRequest struct {
	BaseHeader        BaseHeader
	ApplicationID     ID
	PathID            ID
	FQDNLength        uint16
	CertificateLength uint16
	FQDN              []byte
	Certificate       []byte
	HMAC              HMAC
}

// GenerateDirectionRequest generates Direction Request.
func GenerateDirectionRequest(dr *DirectionRequest, nodeID ID, srcFQDN, distFQDN []byte) {
	err := SerializeBaseHeader(&dr.BaseHeader)
	if err != nil {
		logger.Error(fmt.Errorf("failed to serialize base header: %w", err))
	}

	SerializeType(&dr.BaseHeader, TypeClassDirectionRequest)

	dr.BaseHeader.ID = nodeID

	err = dr.GenerateApplicationID()
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate ApplicationID: %w", err))
	}

	dr.PathID, err = GeneratePathID(srcFQDN, distFQDN)
	if err != nil {
		logger.Error(fmt.Errorf("failed to generate PathID: %w", err))
	}

	dr.FQDNLength = uint16(len(distFQDN))
	dr.FQDN = []byte(distFQDN)

	// FIXME: calculate packet length.
	dr.BaseHeader.MessageLength = 68 + dr.FQDNLength + dr.CertificateLength + 16

	logger.Debug("Generate Direction Request", dr)
}

// GenerateApplicationID is to generate CYPHONIC APP ID.
// APP ID is 128bit length(16 Byte).
func (dr *DirectionRequest) GenerateApplicationID() error {
	// FIXME: ApplicationID is fixed.
	dr.ApplicationID = []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}

	// uuid, err := uuid.NewUUID()
	// if err != nil {
	// 	return err
	// }

	// login.ApplicationID, err = uuid.MarshalBinary()
	// if err != nil {
	// 	return err
	// }

	return nil
}

// GeneratePathID is to generate PathID.
// PathID is used to manage tunnel communication.
func GeneratePathID(srcFQDN, dstFQDN []byte) (b []byte, err error) {
	seed := srcFQDN
	seed = append(seed, dstFQDN...)

	pathID := uuid.NewSHA1(uuid.NameSpaceDNS, seed)

	if b, err = pathID.MarshalBinary(); err != nil {
		return nil, err
	}

	return b, nil
}

// Marshal returns binary data.
func (dr *DirectionRequest) Marshal(key []byte) ([]byte, error) {
	planeBuf := make([]byte, 0, dr.BaseHeader.MessageLength-BaseHeaderLen-HMACLen)
	planeBuffer := bytes.NewBuffer(planeBuf)

	if err := binary.Write(planeBuffer, binary.BigEndian, dr.ApplicationID); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, dr.PathID); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, dr.FQDNLength); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, dr.CertificateLength); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, dr.FQDN); err != nil {
		return nil, err
	}

	if err := binary.Write(planeBuffer, binary.BigEndian, dr.Certificate); err != nil {
		return nil, err
	}

	chiperBuf, err := EncryptPacket(planeBuffer.Bytes(), key)
	if err != nil {
		return nil, err
	}

	dr.BaseHeader.MessageLength = BaseHeaderLen + uint16(len(chiperBuf)) + HMACLen

	buf := make([]byte, 0, dr.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.TransactionID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.Version); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.Flag); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.Type); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.Count); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.SequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.MessageLength); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.NextOpt); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.Reserved); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, dr.BaseHeader.ID); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, chiperBuf); err != nil {
		return nil, err
	}

	dr.HMAC = GenerateHMAC(buffer.Bytes(), GenerateSalt(&dr.BaseHeader))

	if err := binary.Write(buffer, binary.BigEndian, dr.HMAC); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
