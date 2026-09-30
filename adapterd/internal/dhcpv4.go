package internal

import (
	"fmt"
	"sync"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/dhcpv4"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	godhcpv4 "github.com/insomniacslk/dhcp/dhcpv4"
)

// DHCPQueue is the queue of dhcp.
type DHCPQueue struct {
	request []byte
	child   *ChildDevice
}

// QueueDHCPElement is the DHCP element queued to the worker thread.
type QueueDHCPElement struct {
	sync.Mutex
	request []byte
	child   *ChildDevice
	wg      sync.WaitGroup
}

func (adapter *AdapterDevice) NewDHCPElement() *QueueDHCPElement {
	dhcp := adapter.GetDHCPElement()
	dhcp.Mutex = sync.Mutex{}

	return dhcp
}

// clearPointers clears elem fields that contain pointers.
// This makes the garbage collector's life easier and
// avoids accidentally keeping other objects around unnecessarily.
// It also reduces the possible collateral damage from use-after-free bugs.
func (dhcp *QueueDHCPElement) clearPointers() {
	dhcp.request = nil
}

// GetDHCPElement gets QueueDHCPElement.
// Allocate a DHCP element from the pool.
func (adapter *AdapterDevice) GetDHCPElement() *QueueDHCPElement {
	return adapter.pool.dhcpElements.Get().(*QueueDHCPElement)
}

// PutDHCPElement puts QueueDHCPElement.
// Returns the allocated DHCP element to the pool.
func (adapter *AdapterDevice) PutDHCPElement(dhcp *QueueDHCPElement) {
	dhcp.clearPointers()
	adapter.pool.dhcpElements.Put(dhcp)
}

// DHCPstagePacket stages from a DHCP element to the processing queue of the adapter.
func (adapter *AdapterDevice) DHCPstagePacket(dhcp *QueueDHCPElement) {
	for {
		select {
		case adapter.adapterQueue.dhcpStaged <- dhcp:
			return
		default:
		}
		select {
		case tooOld := <-adapter.adapterQueue.dhcpStaged:
			adapter.PutDHCPElement(tooOld)
		default:
		}
	}
}

// DHCPsendStagedPackets stages in the adapter's send queue.
func (adapter *AdapterDevice) DHCPsendStagedPackets() {
	for {
		select {
		case dhcp := <-adapter.adapterQueue.dhcpStaged:
			dhcp.Lock() // Lock for the DHCP packet send
			adapter.adapterQueue.dhcp.c <- dhcp
		default:
			return
		}
	}
}

// RoutineDHCPv4 is handling DHCP Packets.
func (adapter *AdapterDevice) RoutineDHCPv4() {
	defer func() {
		adapter.state.stopping.Done()
		logger.Debug("Routine: DHCPv4 - stopped")
	}()
	logger.Debug("Routine: DHCPv4 - started")

	for dhcp := range adapter.adapterQueue.dhcp.c {
		child := dhcp.child
		payload := dhcp.request

		config := &dhcpv4.DHCPv4Config{
			AdapterL2Addr: adapter.InternalInterface.Card.HardwareAddr,
			AdapterIPv4:   adapter.InternalInterface.Addr4,
			AssignIPv4:    child.VirtualIPv4,
		}

		req, err := dhcpv4.Unmarshal(payload)
		if err != nil {
			logger.Error(fmt.Errorf("failed to unmarshal DHCPv4 request packet: %w", err))
		}

		logger.Info(fmt.Sprintf("RequestType=%v", req.MessageType()))

		resp, respType, err := config.GenerateDHCPv4Response(req)
		if err != nil {
			logger.Error(fmt.Errorf("failed to generate DHCPv4 response packet: %w", err))
		}

		if err = SendEtherPacket(adapter.socket.receiveIPv4, resp); err != nil {
			logger.Error(fmt.Errorf("failed to send DHCPv4 response packet: %w", err))
			child.isVipAssign.Set(false)
		} else {
			logger.Debug(fmt.Sprintf("Send DHCPv4 response packet to %s", child.deviceName))

			// If a DHCPv4 ACK message was sent, set isVipAssign to true.
			// It does nothing when isVipAssign is already true.
			if respType == godhcpv4.MessageTypeAck || child.isVipAssign.Get() {
				child.isVipAssign.Set(true)
				logger.Info(fmt.Sprintf("%s is provisioned", child.deviceName))
			}
		}

		dhcp.wg.Done() // Reporting successful DHCP transaction completion.
		dhcp.Unlock()  // UnLock for the DHCP packet Send
	}
}

// detectChildDeviceWithDHCPRequest detects Child Device from DHCP Request.
func detectChildDeviceWithDHCPRequest(b []byte, adapter *AdapterDevice) (*ChildDevice, error) {
	req, err := dhcpv4.Unmarshal(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal DHCP Request: %w", err)
	}

	child, ok := adapter.children.keyMap[req.ClientHWAddr.String()]
	if !ok {
		return nil, fmt.Errorf("Receive DHCP request packet from %s, but this device is not permitted", req.ClientHWAddr.String())
	}

	logger.Debug(fmt.Sprintf("Receive DHCP request packet from %s [Authentication status=%t]", child.deviceName, child.isLogin.Get()))

	return child, nil
}
