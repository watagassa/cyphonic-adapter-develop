package internal

import (
	"fmt"
	"net"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// DHCPListen returns a Port 67 listened UDP connection.
func DHCPListen() (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", ":67")
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// AdapterDHCPListen listens using the specified port.
func (adapter *AdapterDevice) AdapterDHCPListen(dhcpCh chan DHCPQueue) {
	dhcpv4conn, err := DHCPListen()
	if err != nil {
		logger.Error(fmt.Errorf("failed to read dhcpv4 udp socket=%v: %w", dhcpv4conn.LocalAddr(), err))
	}

	go adapter.ListenDHCPv4Port(dhcpv4conn, dhcpCh) // Port: 67
}

func (adapter *AdapterDevice) ListenDHCPv4Port(c *net.UDPConn, dhcpCh chan DHCPQueue) {
	defer logger.Debug("Routine: Server Listen - stopped")
	logger.Debug("Routine: Server Listen - started")

	for {
		buf := make([]byte, 1500)
		size, addr, err := c.ReadFromUDPAddrPort(buf)
		if err != nil {
			logger.Error(fmt.Errorf("failed to read dhcpv4 udp socket=%v: %w", addr, err))
		}

		child, err := detectChildDeviceWithDHCPRequest(buf[:size], adapter)
		if err != nil {
			continue
		}

		dhcpCh <- DHCPQueue{
			request: buf[:size],
			child:   child,
		}
	}
}
