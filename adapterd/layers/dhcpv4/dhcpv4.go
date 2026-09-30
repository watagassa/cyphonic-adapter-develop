package dhcpv4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/google/gopacket"
	golayers "github.com/google/gopacket/layers"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

const (
	leasePeriod  = 300 // 5m
	renewPeriod  = 150 // 2m30s
	rebindPeriod = 240 // 4m
)

const (
	serverPort = 67 // DHCPv4 server side port
	clientPort = 68 // DHCPv4 client side port
)

type DHCPv4Config struct {
	AdapterL2Addr           net.HardwareAddr
	AdapterIPv4, AssignIPv4 netip.Addr
}

// GenerateDHCPv4Response generates a DHCPv4 response packet based on the given DHCPv4 request packet and configuration.
func (config *DHCPv4Config) GenerateDHCPv4Response(req *dhcpv4.DHCPv4) ([]byte, dhcpv4.MessageType, error) {
	var responseType dhcpv4.MessageType

	var dstIP net.IP

	if req.OpCode != dhcpv4.OpcodeBootRequest {
		return nil, dhcpv4.MessageTypeNone, fmt.Errorf("Unsupported opcode %d. Only BootRequest (%d) is supported", req.OpCode, dhcpv4.OpcodeBootRequest)
	}

	resp, err := dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		return nil, dhcpv4.MessageTypeNone, fmt.Errorf("failed to new DHCPv4 response packet from DHCPv4 request packet: %w", err)
	}

	resp.Options[uint8(dhcpv4.OptionServerIdentifier)] = net.IP(config.AdapterIPv4.AsSlice()).To4()
	resp.Options[uint8(dhcpv4.OptionSubnetMask)] = net.IPv4Mask(255, 255, 0, 0)
	resp.Options[uint8(dhcpv4.OptionBroadcastAddress)] = net.ParseIP(ip.VIPv4Broadcast).To4()
	resp.Options[uint8(dhcpv4.OptionRouter)] = net.IP(config.AdapterIPv4.AsSlice()).To4()
	resp.Options[uint8(dhcpv4.OptionDomainNameServer)] = net.IP(config.AdapterIPv4.AsSlice()).To4()

	leaseTime := make([]byte, 4)
	renewTime := make([]byte, 4)
	rebindTime := make([]byte, 4)
	binary.BigEndian.PutUint32(leaseTime, uint32(leasePeriod))
	binary.BigEndian.PutUint32(renewTime, uint32(renewPeriod))
	binary.BigEndian.PutUint32(rebindTime, uint32(rebindPeriod))

	resp.Options[uint8(dhcpv4.OptionIPAddressLeaseTime)] = leaseTime
	resp.Options[uint8(dhcpv4.OptionRenewTimeValue)] = renewTime
	resp.Options[uint8(dhcpv4.OptionRebindingTimeValue)] = rebindTime

	resp.YourIPAddr = net.IP(config.AssignIPv4.AsSlice()).To4()
	resp.ServerIPAddr = net.IP(config.AdapterIPv4.AsSlice()).To4()

	// When ClientIPAddr is 0.0.0.0, a response packet is generated
	// with the destination IP address as YourIPAddr.
	//
	// If ClientIPAddr is other than 0.0.0.0 (if the previously assigned IP address is alive),
	// a response packet will be generated with the destination IP address as 255.255.255.255.
	if bytes.Equal(req.ClientIPAddr, net.ParseIP(ip.IPv4Unspecified).To4()) {
		resp.ClientIPAddr = req.ClientIPAddr // ClientIPAddr = 0.0.0.0
		dstIP = resp.YourIPAddr              // DstIP = YourIPAddr
	} else {
		resp.ClientIPAddr = resp.YourIPAddr         // ClientIPAddr = YourIPAddr
		dstIP = net.ParseIP(ip.IPv4Broadcast).To4() // DstIP = 255.255.255.255
	}

	switch mt := req.MessageType(); mt {
	case dhcpv4.MessageTypeDiscover:
		resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
		responseType = dhcpv4.MessageTypeOffer
	case dhcpv4.MessageTypeRequest:
		resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
		responseType = dhcpv4.MessageTypeAck
	case dhcpv4.MessageTypeDecline:
		logger.Warn("non-supported message: DHCP DECLINE")
	case dhcpv4.MessageTypeRelease:
		logger.Warn("non-supported message: DHCP RELEASE")
	case dhcpv4.MessageTypeInform:
		logger.Warn("non-supported message: DHCP INFORM")
	default:
		err := errors.New("detect unknown message")
		return nil, dhcpv4.MessageTypeNone, err
	}

	b, err := msgPack(resp, config, dstIP)
	if err != nil {
		return nil, dhcpv4.MessageTypeNone, fmt.Errorf("failed to marshal DHCPv4 response packet: %w", err)
	}

	return b, responseType, nil
}

func msgPack(resp *dhcpv4.DHCPv4, config *DHCPv4Config, dstIP net.IP) ([]byte, error) {
	eth := golayers.Ethernet{
		EthernetType: golayers.EthernetTypeIPv4,
		SrcMAC:       config.AdapterL2Addr,
		DstMAC:       resp.ClientHWAddr,
	}

	ip := golayers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    resp.ServerIPAddr,
		DstIP:    dstIP,
		Protocol: golayers.IPProtocolUDP,
		Flags:    golayers.IPv4DontFragment,
	}
	udp := golayers.UDP{
		SrcPort: serverPort,
		DstPort: clientPort,
	}

	err := udp.SetNetworkLayerForChecksum(&ip)
	if err != nil {
		return nil, fmt.Errorf("failed to set network layer for checksum: %w", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	payload, err := Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DHCPv4 response packet: %w", err)
	}

	// Decode packet
	packet := gopacket.NewPacket(payload, golayers.LayerTypeDHCPv4, gopacket.NoCopy)
	dhcpLayer := packet.Layer(golayers.LayerTypeDHCPv4)
	dhcp, ok := dhcpLayer.(gopacket.SerializableLayer)
	if !ok {
		return nil, fmt.Errorf("layer %s is not serializable", dhcpLayer.LayerType().String())
	}
	err = gopacket.SerializeLayers(buf, opts, &eth, &ip, &udp, dhcp)
	if err != nil {
		return nil, fmt.Errorf("failed to selialize DHCPv4 response packet [SrcIP=%v] [DstIP=%v]: %w", ip.SrcIP, ip.DstIP, err)
	}

	outgoing := buf.Bytes()

	logger.Info(fmt.Sprintf("ResponseType=%v YourIPAddr=%v", resp.MessageType().String(), resp.YourIPAddr))

	return outgoing, nil
}

func Unmarshal(b []byte) (*dhcpv4.DHCPv4, error) {
	req, err := dhcpv4.FromBytes(b)
	if err != nil {
		return nil, fmt.Errorf("failed to convert from bytes: %w", err)
	}

	return req, nil
}

func Marshal(resp *dhcpv4.DHCPv4) ([]byte, error) {
	if resp == nil {
		err := errors.New("dhcp response body is empty")
		return nil, err
	}

	return resp.ToBytes(), nil
}
