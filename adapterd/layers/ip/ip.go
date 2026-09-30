package ip

import (
	"net"
	"net/netip"
)

// IPv4 offset length.
const (
	IPv4offsetTotalLength = 2                           // IPv4offsetPayloadLength is IPv4 offset payload length.
	IPv4offsetSrc         = 12                          // IPv4offsetSrc is IPv6 offset src length.
	IPv4offsetDst         = IPv4offsetSrc + net.IPv4len // IPv4offsetDst is IPv6 offset dst length.
)

// IPv6 offset length.
const (
	IPv6offsetPayloadLength = 4                           // IPv6offsetPayloadLength is IPv6 offset payload length.
	IPv6offsetSrc           = 8                           // IPv6offsetSrc is IPv6 offset src length.
	IPv6offsetDst           = IPv6offsetSrc + net.IPv6len // IPv6offsetDst is IPv6 offset dst length.
)

// Overlay network prefix in CYPHONIC.
const (
	VIPv4PrefixLen    = 16
	VIPv6PrefixLen    = 64
	VIPv4DNSTunPrefix = 24
	VIPv6DNSTunPrefix = 64

	// Define an exact match prefix book that identifies unique hosts.
	// - CYPHONIC virtual IPv4 host match prefix: 32
	// - CYPHONIC virtual IPv6 host match prefix: 128
	VIPv4HostMatchPrefix = 32
	VIPv6HostMatchPrefix = 128

	IPv4Unspecified = "0.0.0.0"
	IPv4Broadcast   = "255.255.255.255"
	VIPv4Prefix     = "198.18.0.0/16"
	VIPv4Netmask    = "255.255.0.0"
	VIPv4Broadcast  = "198.18.255.255"

	IPv6Unspecified = "::"
	VIPv6Prefix     = "2001:0db8:c0ff:ee00::/64"
)

// RFC2375
const (
	AllNodeAddress                     = "ff01::1"
	AllRouterAddress                   = "ff01::2"
	AllNodeMulticastAddress            = "ff02::1"
	AllRouterMulticastAddress          = "ff02::2"
	AllDHCPRelayAgentAndServersAddress = "ff02::1:2"
	AllDHCPServersAddress              = "ff05::1:3"
	AllOSPFv3RouterAddress             = "ff02::5"
	AllOSPFv3DesignatedRouterAddress   = "ff02::6"
	AllRIPngRouterAddress              = "ff02::9"
	AllEIGRPRouterAddress              = "ff02::a"
	AllPIMRouterAddress                = "ff02::d"
	AllSolicitedNodeAddress            = "ff02::1:ff00:0"
)

// IPv4 packet length of each field.
const (
	VersionLength                = 1
	DifferentiatedServicesLength = 1
	TotalLengthLength            = 1
	IdentificationLength         = 2
	OffsetLength                 = 1
	TTLLength                    = 2
	ProtocolLength               = 2
	ChecksumLength               = 2
	SrcIPLength                  = 4
	DstIPLength                  = 4
	IPv4PacketTotalLength        = VersionLength + DifferentiatedServicesLength + TotalLengthLength + IdentificationLength + OffsetLength +
		TTLLength + ProtocolLength + ChecksumLength + SrcIPLength + DstIPLength
)

// IP address lengths (bytes).
const (
	IPv4len = 4
	IPv6len = 16
)

// ZeroAddr4 returns the well-known '0.0.0.0' IPv4 address.
func ZeroAddr4() netip.Addr {
	ipv4zero, _ := netip.AddrFromSlice(net.IPv4zero)
	return ipv4zero.Unmap()
}

// ZeroAddr6 returns the well-known '[::]' IPv6 address.
func ZeroAddr6() netip.Addr {
	ipv6zero, _ := netip.AddrFromSlice(net.IPv6zero)
	return ipv6zero
}

// IsBroadcastAddress reports whether ip is a broadcast address.
func IsBroadcastAddress(ip netip.Addr) bool {
	return ip == netip.MustParseAddr(VIPv4Broadcast)
}

// GetLocalIPv4 returns the non loopback local IPv4 of the host.
// netip.Addr types always return 4 bytes.
func GetLocalIPv4(dnsTun4 netip.Addr) (netip.Addr, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return netip.Addr{}, err // not supported
	}

	for _, address := range addrs {
		if addr4, ok := netip.AddrFromSlice(address.(*net.IPNet).IP.To4()); ok {
			if addr4.Is4() && invalidTUNIPv4(addr4, dnsTun4) && addr4.IsGlobalUnicast() {
				return addr4, nil
			}
		}
	}

	return netip.Addr{}, err
}

// GetLocalIPv6 returns the non loopback local IPv6 of the host.
// netip.Addr types always return 16 bytes.
func GetLocalIPv6(dnsTun6 netip.Addr) (netip.Addr, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return netip.Addr{}, err // not supported
	}

	for _, address := range addrs {
		if addr6, ok := netip.AddrFromSlice(address.(*net.IPNet).IP.To16()); ok {
			if !addr6.Is4In6() && invalidTUNIPv6(addr6, dnsTun6) && addr6.IsGlobalUnicast() {
				return addr6, nil
			}
		}
	}

	return netip.Addr{}, err
}

// invalidTUNIPv4 compares the IPv4 addresses received as arguments.
// and, returns a bool.
func invalidTUNIPv4(targetIP, dnsTun4 netip.Addr) bool {
	prefix := netip.MustParsePrefix(VIPv4Prefix)
	pf := netip.PrefixFrom(targetIP, VIPv4PrefixLen)

	if (targetIP != dnsTun4) && !(prefix.Overlaps(pf)) {
		return true
	}

	return false
}

// invalidTUNIPv6 compares the IPv6 addresses received as arguments.
// and, returns a bool.
func invalidTUNIPv6(targetIP, dnsTun6 netip.Addr) bool {
	prefix := netip.MustParsePrefix(VIPv6Prefix)
	pf := netip.PrefixFrom(targetIP, VIPv6PrefixLen)

	if (targetIP != dnsTun6) && !(prefix.Overlaps(pf)) {
		return true
	}

	return false
}
