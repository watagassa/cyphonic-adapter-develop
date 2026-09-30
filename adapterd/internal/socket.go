package internal

import (
	"fmt"
	"net"
	"syscall"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// htons (Host to Network Short) function converts a 16-bit unsigned integer
// from host byte order to network byte order (big-endian).
// It swaps the byte order from little-endian to big-endian or vice versa.
func htons(host uint16) uint16 {
	return (host&0xff)<<8 | (host >> 8)
}

// RecvIPv4RawSocket creates a raw socket for receiving IPv4 packet.
func RecvIPv4RawSocket(intfIndex *net.Interface) (int, error) {
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_IP)))
	if err != nil {
		return -1, err
	}

	addr := syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  intfIndex.Index,
	}

	if err := syscall.Bind(fd, &addr); err != nil {
		return -1, err
	}

	return fd, nil
}

// SendIPv4RawSocket creates a raw socket for sending IPv4 packet.
func SendIPv4RawSocket() (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return -1, err
	}

	ip := net.ParseIP(ip.IPv4Unspecified).To4()
	addr := syscall.SockaddrInet4{
		Addr: [4]byte{ip[0], ip[1], ip[2], ip[3]},
	}

	if err = syscall.Bind(fd, &addr); err != nil {
		return -1, err
	}

	return fd, nil
}

// RecvIPv6RawSocket creates a raw socket for receiving IPv6 packet.
func RecvIPv6RawSocket(intfIndex *net.Interface) (int, error) {
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_IPV6)))
	if err != nil {
		return -1, err
	}

	addr := syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  intfIndex.Index,
	}

	if err := syscall.Bind(fd, &addr); err != nil {
		return -1, err
	}

	// Received in promiscuous mode
	if err := syscall.SetLsfPromisc(intfIndex.Name, true); err != nil {
		return -1, err
	}

	return fd, nil
}

// SendIPv6RawSocket creates a raw socket for sending IPv6 packet.
func SendIPv6RawSocket() (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return -1, err
	}

	ip := net.ParseIP("::").To16()
	addr := syscall.SockaddrInet6{
		Addr: [16]byte{ip[0], ip[1], ip[2], ip[3], ip[4], ip[5], ip[6], ip[7], ip[8], ip[9], ip[10], ip[11], ip[12], ip[13], ip[14], ip[15]},
	}

	if err = syscall.Bind(fd, &addr); err != nil {
		return -1, err
	}

	return fd, nil
}

// SendEtherPacket uses the generated file descriptor to send Ether Layer packets.
func SendEtherPacket(fd int, b []byte) error {
	if _, err := syscall.Write(fd, b); err != nil {
		return err
	}

	return nil
}

// SendPacket4 sends IPv4 packet.
func SendPacket4(s int, b []byte, ip []byte) error {
	addr := syscall.SockaddrInet4{
		Addr: [4]byte{ip[0], ip[1], ip[2], ip[3]},
	}

	if err := syscall.Sendto(s, b, 0, &addr); err != nil {
		return err
	}

	return nil
}

// SendPacket6 sends IPv6 packet.
func SendPacket6(s int, b []byte, ip []byte) error {
	addr := syscall.SockaddrInet6{
		Addr: [16]byte{ip[0], ip[1], ip[2], ip[3], ip[4], ip[5], ip[6], ip[7], ip[8], ip[9], ip[10], ip[11], ip[12], ip[13], ip[14], ip[15]},
	}

	if err := syscall.Sendto(s, b, 0, &addr); err != nil {
		return err
	}

	return nil
}

func (adapter *AdapterDevice) CreateDescriptor(netInterface *net.Interface) {
	var err error

	// Receive IPv4 RawSocket
	adapter.socket.receiveIPv4, err = RecvIPv4RawSocket(netInterface)
	if err != nil {
		logger.Error(fmt.Errorf("failed to create receiving IPv4 raw socket: %w", err))
	}

	// Receive IPv6 RawSocket
	adapter.socket.receiveIPv6, err = RecvIPv6RawSocket(netInterface)
	if err != nil {
		logger.Error(fmt.Errorf("failed to create receiving IPv6 raw socket: %w", err))
	}

	// Send IPv4 RawSocket
	adapter.socket.sendIPv4, err = SendIPv4RawSocket()
	if err != nil {
		logger.Error(fmt.Errorf("failed to create sending IPv4 raw socket: %w", err))
	}

	// Send IPv6 RawSocket
	adapter.socket.sendIPv6, err = SendIPv6RawSocket()
	if err != nil {
		logger.Error(fmt.Errorf("failed to create sending IPv6 raw socket: %w", err))
	}
}
