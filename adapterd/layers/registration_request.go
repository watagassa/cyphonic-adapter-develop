// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
)

// RegistrationRequest is Registration Request of a CYPHONIC packet.
type RegistrationRequest struct {
	BaseHeader       BaseHeader
	ApplicationPort  uint16
	NotificationType TypeNotificationClass
	NodeIPv4Address  netip.Addr
	NodeIPv6Address  netip.Addr
	HMAC             HMAC
}

// TypeNotificationClass defines the class associated with Notification type in CYPHONIC packets.
type TypeNotificationClass uint16

// TypeNotificationClass knowns values.
const (
	TypeNotificationNone                TypeNotificationClass = 0
	TypeFirebaseCloudMessaging          TypeNotificationClass = 1
	TypeApplePushNotificationService    TypeNotificationClass = 2
	TypeWindowsPushNotificationsService TypeNotificationClass = 3
)

// TypeLocalIPVersion defines to identify the local IP version of noded.
type TypeLocalIPVersion uint8

// TypeLocalIPVersion knowns values.
const (
	TypeLocalIPNone           TypeLocalIPVersion = 0
	TypeLocalIPVersion4       TypeLocalIPVersion = 1
	TypeLocalIPVersion6       TypeLocalIPVersion = 2
	TypeLocalDualStackNetwork TypeLocalIPVersion = 3
)

// Payload length in RegistrationRequest.
const (
	// ApplicationPortSize lengths (bytes).
	ApplicationPortSize = 2

	// NotificationTypeSize lengths (bytes).
	NotificationTypeSize = 2

	// NodeIPv4AddressSize lengths (bytes).
	NodeIPv4AddressSize = 4

	// NodeIPv6AddressSize lengths (bytes).
	NodeIPv6AddressSize = 16
)

// BaseRegistrationRequestLength is length of (base) body part of RegistrationRequest.
const BaseRegistrationRequestLength = BaseHeaderLen + ApplicationPortSize + NotificationTypeSize + NodeIPv4AddressSize + NodeIPv6AddressSize

// GenerateRegistrationRequest generates registration request packet.
// It returns a RegistrationRequest for Node Management Service.
func GenerateRegistrationRequest(request *RegistrationRequest, nodeID ID, local net.Addr, dnsTun4, dnsTun6 netip.Addr) (localIPVer TypeLocalIPVersion, err error) {
	if nodeInfo, ok := local.(*net.UDPAddr); ok {
		if err := SerializeBaseHeader(&request.BaseHeader); err != nil {
			return TypeLocalIPNone, fmt.Errorf("failed to serialize BaseHeader: %w", err)
		}

		SerializeType(&request.BaseHeader, TypeClassRegistrationRequest)
		SerializeFlag(&request.BaseHeader, FlagClassFirstRegistration)

		// FIXME: calculate packet length.
		request.BaseHeader.MessageLength = 72
		request.BaseHeader.ID = nodeID
		serializeApplicationPort(request, nodeInfo.Port)
		serializeNotificationType(request)

		localIPVer, err = serializeNodeAddress(request, dnsTun4, dnsTun6)
		if err != nil {
			return TypeLocalIPNone, fmt.Errorf("failed to serialize node address: %w", err)
		}

		// FIXME: HMAC is fixed.
		request.HMAC = []byte{
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
		}

		return localIPVer, nil
	}

	err = errors.New("unknown local IP version")

	return TypeLocalIPNone, fmt.Errorf("failed to serialize node address: %w", err)
}

// SerializeRegistrationRequestPacketLength sets the length of the Registration Request packet.
func SerializeRegistrationRequestPacketLength(base *BaseHeader) uint16 {
	base.MessageLength = BaseRegistrationRequestLength + HMACLen

	return base.MessageLength
}

// serializeApplicationPort sets the application port of the CYPHONIC Node.
func serializeApplicationPort(request *RegistrationRequest, port int) uint16 {
	request.ApplicationPort = uint16(port)

	return request.ApplicationPort
}

// serializeNotificationType sets the notification type of the CYPHONIC Node.
func serializeNotificationType(request *RegistrationRequest) TypeNotificationClass {
	// TODO: Set notification type in config file (config.yaml).
	request.NotificationType = TypeNotificationNone

	return request.NotificationType
}

// serializeNodeAddress sets the address of the CYPHONIC Node.
func serializeNodeAddress(request *RegistrationRequest, dnsTun4, dnsTun6 netip.Addr) (TypeLocalIPVersion, error) {
	ipv4, err := ip.GetLocalIPv4(dnsTun4)
	if err != nil {
		return TypeLocalIPNone, fmt.Errorf("failed to get virtual IPv4 address: %w", err)
	}

	ipv6, err := ip.GetLocalIPv6(dnsTun6)
	if err != nil {
		return TypeLocalIPNone, fmt.Errorf("failed to get virtual IPv6 address: %w", err)
	}

	switch {
	case ipv4 != netip.Addr{} && ipv6 != netip.Addr{}:
		request.NodeIPv4Address = ipv4
		request.NodeIPv6Address = ipv6

		return TypeLocalDualStackNetwork, nil
	case ipv4 != netip.Addr{} && ipv6 == netip.Addr{}:
		request.NodeIPv4Address = ipv4
		request.NodeIPv6Address = ip.ZeroAddr6()

		return TypeLocalIPVersion4, nil
	case ipv4 == netip.Addr{} && ipv6 != netip.Addr{}:
		request.NodeIPv4Address = ip.ZeroAddr4()
		request.NodeIPv6Address = ipv6

		return TypeLocalIPVersion6, nil
	default:
		return TypeLocalIPNone, fmt.Errorf("failed to get local IP version: %w", err)
	}
}

// Marshal returns binary data.
func (rreq *RegistrationRequest) Marshal(key []byte) ([]byte, error) {
	planeBuf := make([]byte, 0, rreq.BaseHeader.MessageLength-BaseHeaderLen-HMACLen)
	planeBuffer := bytes.NewBuffer(planeBuf)

	err := binary.Write(planeBuffer, binary.BigEndian, rreq.ApplicationPort)
	if err != nil {
		return nil, err
	}

	err = binary.Write(planeBuffer, binary.BigEndian, rreq.NotificationType)
	if err != nil {
		return nil, err
	}

	err = binary.Write(planeBuffer, binary.BigEndian, rreq.NodeIPv4Address.As4())
	if err != nil {
		return nil, err
	}

	err = binary.Write(planeBuffer, binary.BigEndian, rreq.NodeIPv6Address.As16())
	if err != nil {
		return nil, err
	}

	chiperBuf, err := EncryptPacket(planeBuffer.Bytes(), key)
	if err != nil {
		return nil, err
	}

	rreq.BaseHeader.MessageLength = BaseHeaderLen + uint16(len(chiperBuf)) + HMACLen

	buf := make([]byte, 0, rreq.BaseHeader.MessageLength)
	buffer := bytes.NewBuffer(buf)

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.TransactionID)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.Version)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.Flag)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.Type)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.Count)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.SequenceNumber)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.MessageLength)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.NextOpt)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.Reserved)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, rreq.BaseHeader.ID)
	if err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, chiperBuf); err != nil {
		return nil, err
	}

	rreq.HMAC = GenerateHMAC(buffer.Bytes(), GenerateSalt(&rreq.BaseHeader))

	err = binary.Write(buffer, binary.BigEndian, rreq.HMAC)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
