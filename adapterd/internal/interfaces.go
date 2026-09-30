package internal

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
)

// Maximum Transmission Unit (MTU) is the largest size frame or packet.
const MTU = 1500

// SetInternalInterface sets the address of the adapter's internal network interface.
// Multiple general nodes are connected to the internal interface.
func (adapter *AdapterDevice) SetInternalInterface() error {
	netInterface, err := net.InterfaceByName(adapter.Cfg.Adapterd.InternalInterface)
	if err != nil {
		return fmt.Errorf("failed to find adapter internal interface (%v) addresses: %w", adapter.Cfg.Adapterd.InternalInterface, err)
	}

	adapter.InternalInterface.Card = netInterface

	addrs, err := netInterface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get adapter internal interface addresses: %w", err)
	}

	for _, addr := range addrs {
		addr := netip.MustParseAddr(addr.String()[:strings.Index(addr.String(), "/")])

		prefix4 := netip.MustParsePrefix(ip.VIPv4Prefix)
		pf4 := netip.PrefixFrom(addr, ip.VIPv4PrefixLen)

		prefix6 := netip.MustParsePrefix(ip.VIPv6Prefix)
		pf6 := netip.PrefixFrom(addr, ip.VIPv6PrefixLen)

		// adapter virtual IPv4 address
		if prefix4.Overlaps(pf4) {
			adapter.InternalInterface.Addr4 = addr
		}

		// adapter virtual IPv6 address
		if prefix6.Overlaps(pf6) {
			adapter.InternalInterface.Addr6 = addr
		}

		// adapter link local unicast address
		if addr.IsLinkLocalUnicast() {
			adapter.InternalInterface.LinkLocalAddr = addr
		}
	}

	// whether ip address is not invalid.
	if !adapter.InternalInterface.Addr4.IsValid() || !adapter.InternalInterface.Addr6.IsValid() || !adapter.InternalInterface.LinkLocalAddr.IsValid() {
		return fmt.Errorf("internal Interface addr is invalid [Addr4=%v] [Addr6=%v] [LinkLocalAddr=%v]", adapter.InternalInterface.Addr4, adapter.InternalInterface.Addr6, adapter.InternalInterface.LinkLocalAddr)
	}

	return nil
}
