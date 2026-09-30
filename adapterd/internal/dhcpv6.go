package internal

import (
	"encoding/binary"
	"fmt"
	"net/netip"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/dhcpv6"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	godhcpv6 "github.com/insomniacslk/dhcp/dhcpv6"
)

// RoutineDHCPv6 is handling DHCPv6 Packets.
func (adapter *AdapterDevice) RoutineDHCPv6() {
	defer func() {
		adapter.state.stopping.Done()
		logger.Debug("Routine: DHCPv6 - stopped")
	}()
	logger.Debug("Routine: DHCPv6 - started")

	for dhcp := range adapter.adapterQueue.dhcp.c {
		child := dhcp.child
		payload := dhcp.request

		child.addrMu.RLock()
		childLinkLocalAddr := child.LinkLocalAddr
		child.addrMu.RUnlock()

		if !childLinkLocalAddr.IsValid() {
			logger.Error(fmt.Errorf("failed to get Link Local Address for child %s", child.deviceName))
			continue
		}

		config := &dhcpv6.DHCPv6Config{
			AdapterL2Addr: adapter.InternalInterface.Card.HardwareAddr,
			ChildL2Addr:   child.MacAddress,
			AdapterIPv6:   adapter.InternalInterface.LinkLocalAddr,
			ChildIPv6:     childLinkLocalAddr,
			AssignIPv6:    child.VirtualIPv6,
		}

		req, err := godhcpv6.MessageFromBytes(payload)
		if err != nil {
			logger.Error(fmt.Errorf("failed to unmarshal DHCPv6 packet: %w", err))
			continue
		}

		logger.Info(fmt.Sprintf("RequestType=%v", req.Type()))

		if req.Type() == godhcpv6.MessageTypeSolicit && req.GetOneOption(godhcpv6.OptionRapidCommit) == nil {
			resp, _, err := config.GenerateDHCPv6Advertise(req)
			if err != nil {
				logger.Error(fmt.Errorf("failed to generate DHCPv6 Advertise packet: %w", err))
				continue
			}

			if err = SendEtherPacket(adapter.socket.receiveIPv6, resp); err != nil {
				logger.Error(fmt.Errorf("failed to send DHCPv6 Advertise packet: %w", err))
				child.isVipAssign.Set(false)
			} else {
				logger.Debug(fmt.Sprintf("Send DHCPv6 Advertise packet to %s", child.deviceName))
			}
		} else {
			resp, respType, err := config.GenerateDHCPv6Reply(req)
			if err != nil {
				logger.Error(fmt.Errorf("failed to generate DHCPv6 Reply packet: %w", err))
				continue
			}

			if err = SendEtherPacket(adapter.socket.receiveIPv6, resp); err != nil {
				logger.Error(fmt.Errorf("failed to send DHCPv6 Reply packet: %w", err))
				child.isVipAssign.Set(false)
			} else {
				logger.Debug(fmt.Sprintf("Send DHCPv6 Reply packet to %s", child.deviceName))
				if respType == godhcpv6.MessageTypeReply || child.isVipAssign.Get() {
					child.isVipAssign.Set(true)
					logger.Info(fmt.Sprintf("%s is provisioned", child.deviceName))
				}
			}
		}

		dhcp.wg.Done() // Reporting successful DHCP transaction completion.
		dhcp.Unlock()  // UnLock for the DHCP packet Send
	}
}

func detectChildDeviceWithDHCPv6Request(b []byte, srcAddrPort netip.AddrPort, adapter *AdapterDevice) ([]byte, *ChildDevice, error) {
	buf, err := fixVendorClassOption(b)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fix vendor class option: %w", err)
	}

	srcLinkLocal := srcAddrPort.Addr().WithZone("")

	child := adapter.findChildByLinkLocalAddress(srcLinkLocal)
	if child == nil {
		return nil, nil, fmt.Errorf("failed to find child device for Link Local Address: %v", srcLinkLocal)
	}

	logger.Debug(fmt.Sprintf("Receive DHCPv6 request packet from %s [Authentication status=%t]", child.deviceName, child.isLogin.Get()))

	return buf, child, nil
}

// findChildByLinkLocalAddress finds a child device by its Link-Local Address.
func (adapter *AdapterDevice) findChildByLinkLocalAddress(linkLocal netip.Addr) *ChildDevice {
	adapter.children.RLock()
	defer adapter.children.RUnlock()

	for _, child := range adapter.children.keyMap {
		if child.LinkLocalAddr == linkLocal {
			return child
		}
	}
	return nil
}

// fixVendorClassOption pads malformed Vendor Class options (Option 16) that lack
// the vendor-class-data length field, which causes the DHCPv6 library to fail on parse.
// See RFC 8415 Section 21.16.
func fixVendorClassOption(packet []byte) ([]byte, error) {
	const (
		dhcpv6HeaderLen   = 4  // msg-type (1) + transaction-id (3)
		optionHeaderLen   = 4  // option-code (2) + option-len (2)
		optVendorClass    = 16 // OPTION_VENDOR_CLASS
		minVendorClassLen = 6  // enterprise-number (4) + vendor-class-data-len (2)
	)

	if len(packet) < dhcpv6HeaderLen {
		return nil, fmt.Errorf("DHCPv6 packet too short: %d bytes", len(packet))
	}

	offset := dhcpv6HeaderLen
	for offset+optionHeaderLen <= len(packet) {
		optCode := binary.BigEndian.Uint16(packet[offset : offset+2])
		optLen := int(binary.BigEndian.Uint16(packet[offset+2 : offset+4]))

		optEnd := offset + optionHeaderLen + optLen
		if optEnd > len(packet) {
			return nil, fmt.Errorf("option %d (len=%d) exceeds packet boundary", optCode, optLen)
		}

		if optCode == optVendorClass && optLen < minVendorClassLen {
			padLen := minVendorClassLen - optLen

			result := make([]byte, 0, len(packet)+padLen)
			result = append(result, packet[:optEnd]...)
			result = append(result, make([]byte, padLen)...)
			result = append(result, packet[optEnd:]...)

			// Update the option length to the padded size.
			binary.BigEndian.PutUint16(result[offset+2:offset+4], minVendorClassLen)

			return result, nil
		}

		offset = optEnd
	}

	return packet, nil
}
