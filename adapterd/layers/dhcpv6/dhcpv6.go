package dhcpv6

import (
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/google/gopacket"
	golayers "github.com/google/gopacket/layers"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/iana"
)

type DHCPv6Config struct {
	AdapterL2Addr, ChildL2Addr         net.HardwareAddr
	AdapterIPv6, ChildIPv6, AssignIPv6 netip.Addr
}

const (
	validPeriod     = 300 * time.Second // 5m
	preferredPeriod = 270 * time.Second // 4m30s
	renewPeriod     = 150 * time.Second // 2m30s
	rebindPeriod    = 240 * time.Second // 4m
)

const (
	serverPort = 547 // DHCPv6 server side port
	clientPort = 546 // DHCPv6 client side port
)

// GenerateDHCPv6Advertise generates a DHCPv6 Advertise message based on the given DHCPv6 Solicit message and configuration.
func (config *DHCPv6Config) GenerateDHCPv6Advertise(req *dhcpv6.Message) ([]byte, dhcpv6.MessageType, error) {
	logger.Debug("Generate DHCPv6 Advertise Message")

	resp, err := dhcpv6.NewAdvertiseFromSolicit(req)
	if err != nil {
		logger.Error(fmt.Errorf("failed to create DHCPv6 Advertise message: %w", err))
		return nil, dhcpv6.MessageTypeNone, err
	}

	iface, err := net.InterfaceByName("eth1")
	if err != nil {
		logger.Error(fmt.Errorf("failed to get interface: %w", err))
		return nil, dhcpv6.MessageTypeNone, err
	}

	duid := &dhcpv6.DUIDLLT{
		HWType:        iana.HWTypeEthernet,
		Time:          dhcpv6.GetTime(),
		LinkLayerAddr: iface.HardwareAddr,
	}

	ianaOpt := req.GetOneOption(dhcpv6.OptionIANA)
	if ianaOpt == nil {
		logger.Error(fmt.Errorf("failed to get IANA option"))
		return nil, dhcpv6.MessageTypeNone, err
	}

	iaAddr := &dhcpv6.OptIAAddress{
		IPv6Addr:          config.AssignIPv6.AsSlice(),
		PreferredLifetime: preferredPeriod,
		ValidLifetime:     validPeriod,
	}

	iana := &dhcpv6.OptIANA{
		IaId: ianaOpt.(*dhcpv6.OptIANA).IaId,
		T1:   renewPeriod,
		T2:   rebindPeriod,
		Options: dhcpv6.IdentityOptions{
			Options: []dhcpv6.Option{iaAddr},
		},
	}

	dns := dhcpv6.OptDNS(net.ParseIP("2001:db8:c0ff:ee00::1"))

	resp.AddOption(dhcpv6.OptServerID(duid))
	resp.AddOption(iana)
	resp.AddOption(dns)

	b, err := msgv6Pack(resp, config)
	if err != nil {
		logger.Error(fmt.Errorf("failed to marshal DHCPv6 advertise packet: %w", err))
		return nil, dhcpv6.MessageTypeNone, err
	}

	return b, resp.Type(), nil
}

// GenerateDHCPv6Reply generates a DHCPv6 reply message based on the given DHCPv6 request message and configuration.
func (config *DHCPv6Config) GenerateDHCPv6Reply(req *dhcpv6.Message) ([]byte, dhcpv6.MessageType, error) {
	logger.Debug("Generate DHCPv6 Reply Message")

	resp, err := dhcpv6.NewReplyFromMessage(req)
	if err != nil {
		logger.Error(fmt.Errorf("failed to create DHCPv6 Reply message: %w", err))
		return nil, dhcpv6.MessageTypeNone, err
	}

	iface, err := net.InterfaceByName("eth1")
	if err != nil {
		logger.Error(fmt.Errorf("failed to get interface: %w", err))
		return nil, dhcpv6.MessageTypeNone, err
	}

	duid := &dhcpv6.DUIDLLT{
		HWType:        iana.HWTypeEthernet,
		Time:          dhcpv6.GetTime(),
		LinkLayerAddr: iface.HardwareAddr,
	}

	ianaOpt := req.GetOneOption(dhcpv6.OptionIANA)
	if ianaOpt == nil {
		logger.Error(fmt.Errorf("failed to get IANA option"))
		return nil, dhcpv6.MessageTypeNone, err
	}

	iaAddr := &dhcpv6.OptIAAddress{
		IPv6Addr:          config.AssignIPv6.AsSlice(),
		PreferredLifetime: preferredPeriod,
		ValidLifetime:     validPeriod,
	}

	iana := &dhcpv6.OptIANA{
		IaId: ianaOpt.(*dhcpv6.OptIANA).IaId,
		T1:   renewPeriod,
		T2:   rebindPeriod,
		Options: dhcpv6.IdentityOptions{
			Options: []dhcpv6.Option{iaAddr},
		},
	}

	dns := dhcpv6.OptDNS(net.ParseIP("2001:db8:c0ff:ee00::1"))

	resp.AddOption(dhcpv6.OptClientID(req.Options.ClientID()))
	resp.AddOption(dhcpv6.OptServerID(duid))
	resp.AddOption(iana)
	resp.AddOption(dns)

	b, err := msgv6Pack(resp, config)
	if err != nil {
		logger.Error(fmt.Errorf("failed to marshal DHCPv6 response packet: %w", err))
		return nil, dhcpv6.MessageTypeNone, err
	}

	return b, resp.Type(), nil
}

func msgv6Pack(resp *dhcpv6.Message, config *DHCPv6Config) ([]byte, error) {
	eth := golayers.Ethernet{
		EthernetType: golayers.EthernetTypeIPv6,
		SrcMAC:       config.AdapterL2Addr,
		DstMAC:       config.ChildL2Addr,
	}

	ip := golayers.IPv6{
		Version:    6,
		NextHeader: golayers.IPProtocolUDP,
		HopLimit:   64,
		SrcIP:      config.AdapterIPv6.AsSlice(),
		DstIP:      config.ChildIPv6.AsSlice(),
	}

	udp := golayers.UDP{
		SrcPort: serverPort,
		DstPort: clientPort,
	}

	err := udp.SetNetworkLayerForChecksum(&ip)
	if err != nil {
		logger.Error(fmt.Errorf("failed to set network layer for checksum: %w", err))
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	payload := resp.ToBytes()

	packet := gopacket.NewPacket(payload, golayers.LayerTypeDHCPv6, gopacket.NoCopy)
	dhcpLayer := packet.Layer(golayers.LayerTypeDHCPv6)
	dhcp, ok := dhcpLayer.(gopacket.SerializableLayer)
	if !ok {
		logger.Error(fmt.Errorf("layer %s is not serializable", dhcpLayer.LayerType().String()))
	}
	err = gopacket.SerializeLayers(buf, opts, &eth, &ip, &udp, dhcp)
	if err != nil {
		logger.Error(fmt.Errorf("failed to serialize DHCPv6 response packet [SrcIP=%v] [DstIP=%v]: %w", ip.SrcIP, ip.DstIP, err))
	}

	outgoing := buf.Bytes()

	logger.Info(fmt.Sprintf("ResponseType=%v YourIPAddr=%v", resp.MessageType, config.AssignIPv6))

	return outgoing, nil
}
