package internal

import (
	"fmt"
	"net"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"golang.org/x/net/ipv6"
)

// DHCPv6Listen returns a Port 547 listened UDP connection for DHCPv6.
func DHCPv6Listen() (*net.UDPConn, error) {
	iface, err := net.InterfaceByName("eth1")
	if iface == nil {
		return nil, err
	}

	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP(ip.AllDHCPRelayAgentAndServersAddress),
		Port: 547,
	}

	conn, err := net.ListenUDP("udp6", udpAddr)

	p := ipv6.NewPacketConn(conn)
	if err := p.JoinGroup(iface, udpAddr); err != nil {
		return nil, err
	}

	return conn, nil
}

// AdapterListen listens using the specified port.
func (adapter *AdapterDevice) AdapterDHCPv6Listen(dhcpCh chan DHCPQueue) {
	dhcpv6conn, err := DHCPv6Listen()
	if err != nil {
		logger.Error(fmt.Errorf("failed to read dhcpv6 udp socket=%v %w", dhcpv6conn.LocalAddr(), err))
		return
	}

	go adapter.ListenDHCPv6Port(dhcpv6conn, dhcpCh) // Port: 547
}

func (adapter *AdapterDevice) ListenDHCPv6Port(c *net.UDPConn, dhcpCh chan DHCPQueue) {
	defer logger.Debug("Routine: DHCPv6 Listen - stopped")
	logger.Debug("Routine: DHCPv6 Listen - started")

	for {
		buf := make([]byte, 1500)
		size, addr, err := c.ReadFromUDPAddrPort(buf)
		if err != nil {
			logger.Error(fmt.Errorf("failed to read udp socket=%v: %w", addr, err))
		}

		b, child, err := detectChildDeviceWithDHCPv6Request(buf[:size], addr, adapter)
		if err != nil {
			logger.Error(fmt.Errorf("failed to detect child device: %w", err))
			continue
		}

		dhcpCh <- DHCPQueue{
			request: b,
			child:   child,
		}
	}
}
