package internal

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/internal/cache"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/cmd"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// childState returns device.state.state as a deviceState
// See those docs for how to interpret this value.
func (child *ChildDevice) childState() deviceState {
	return deviceState(atomic.LoadUint32(&child.state.state))
}

// isDown reports whether the child device is not configured (or is attempting to come up).
// See device.state.state comments for how to interpret this value.
func (child *ChildDevice) isDown() bool {
	return child.childState() == deviceStateDown
}

// isUp reports whether the child device is up (or is attempting to come up).
// See device.state.state comments for how to interpret this value.
func (child *ChildDevice) isUp() bool {
	return child.childState() == deviceStateUp
}

// isClosed reports whether the child device is closed (or is closing).
// See device.state.state comments for how to interpret this value.
func (child *ChildDevice) isClosed() bool {
	return child.childState() == deviceStateClosed
}

// ChildDevice manages each child device.
type ChildDevice struct {
	sync.RWMutex

	state struct {
		// state holds the device's state. It is accessed atomically.
		// Use the device.deviceState method to read it.
		// device.deviceState does not acquire the mutex, so it captures only a snapshot.
		// During state transitions, the state variable is updated before the device itself.
		// The state is thus either the current state of the device or
		// the intended future state of the device.
		// For example, while executing a call to Up, state will be deviceStateUp.
		// There is no guarantee that that intended future state of the device
		// will become the actual state; Up can fail.
		// The device can also change state multiple times between time of check and time of use.
		// Unsynchronized uses of state must therefore be advisory/best-effort only.
		state uint32 // actually a deviceState, but typed uint32 for convenience
		// stopping blocks until all inputs to Device have been closed.
		stopping sync.WaitGroup
		// mu protects state changes.
		sync.Mutex
	}

	// After receiving the Registration Response, put the ChildDevice into the running state.
	running sync.WaitGroup

	// Cancels the generated routine (Routine between adapter device and child device) if FUGA exits.
	routineCancel context.CancelFunc

	Adapter *AdapterDevice // connected adapter information

	MacAddress    net.HardwareAddr
	LinkLocalAddr netip.Addr
	isVirtualIPv6 AtomicBool
	VirtualIPv4   netip.Addr
	VirtualIPv6   netip.Addr

	addrMu sync.RWMutex

	isLogin     AtomicBool // child device authentication status
	isVipAssign AtomicBool // virtual IP address assignment status

	nodeID    []byte
	fqdn      []byte
	commonKey []byte

	peers struct {
		sync.RWMutex
		keyMap map[string]*Peer
	}

	tunnelCache *TunnelInfoCache

	allowedips AllowedIPs

	pool struct {
		messageBuffers   *WaitPool
		inboundElements  *WaitPool
		outboundElements *WaitPool
	}

	deviceName      string
	deviceID        []byte
	hashedPassword  [64]byte
	authType        int
	certificatePath string
}

// child device's liveness probe parameter.
const (
	ChildDeviceLivenessProbeInterval = time.Minute * 3
	ChildDeviceLivenessProbeCount    = 3
)

// NewChildDevice initializes the child device.
func (adapter *AdapterDevice) NewChildDevice(dhcpMsg DHCPQueue, nodeIDCache *cache.NodeIDCache) {
	child := dhcpMsg.child
	dhcpRequest := dhcpMsg.request
	dhcp := adapter.NewDHCPElement()

	child.Lock()
	defer child.Unlock()

	if !child.isLogin.Get() {
		child.state.state = uint32(deviceStateDown)

		// TODO: Implementation of IEEE 802.1X authentication.

		connTLS, err := adapter.MakeTLSConn()
		if err != nil {
			logger.Error(fmt.Errorf("failed to establish tls connection: %w", err))
		}

		authElem, err := layers.SetAuthenticationElement(child.authType, child.deviceID, child.hashedPassword, child.certificatePath)
		if err != nil {
			logger.Error(fmt.Errorf("failed to set authentication element: %w", err))
		}

		/* Authentication process */
		logger.Debug(fmt.Sprintf("Execute authentication process=%s", string(authElem.DeviceID)))

		lres := HandleAuthentication(connTLS, authElem)
		logger.Debug(fmt.Sprintf("Successfully executed %s's authentication process", child.deviceName))

		child.fqdn = lres.FQDN            // child device's FQDN
		child.nodeID = lres.BaseHeader.ID // child device's nodeID
		child.VirtualIPv4 = lres.VirtualIPv4Address
		child.VirtualIPv6 = lres.VirtualIPv6Address
		child.commonKey = lres.CommonKey // child device - NMS common key

		logger.Info(fmt.Sprintf("ChildDevice - DeviceName=%s", child.deviceName))
		logger.Info(fmt.Sprintf("ChildDevice - NodeID=%x", lres.BaseHeader.ID))
		logger.Info(fmt.Sprintf("ChildDevice - FQDN=%s", string(lres.FQDN)))
		logger.Info(fmt.Sprintf("ChildDevice - VirtualIPv4=%v", lres.VirtualIPv4Address))
		logger.Info(fmt.Sprintf("ChildDevice - VirtualIPv6=%v", lres.VirtualIPv6Address))

		// Save virtual IP information to Redis
		if adapter.redisClient != nil {
			if err := adapter.redisClient.SaveVirtualIP(
				child.deviceName,
				string(child.deviceID),
				child.MacAddress.String(),
				lres.VirtualIPv4Address,
				lres.VirtualIPv6Address,
			); err != nil {
				logger.Error(fmt.Errorf("failed to save virtual IP information to Redis: %w", err))
			} else {
				logger.Debug(fmt.Sprintf("Successfully saved virtual IP information to Redis for %s", child.deviceName))
			}
		}

		child.isLogin.Set(true)

		/* Registration process */
		logger.Debug(fmt.Sprintf("Execute registration process=%x", child.nodeID))

		if err := child.HandleChildDeviceRegistration(nodeIDCache); err != nil {
			logger.Error(fmt.Errorf("failed to execute child device's registration process: %w", err))
		} else {
			logger.Debug(fmt.Sprintf("Successfully executed %s's registration process", child.deviceName))
			child.running.Add(1) // add wainting queue
		}
	}

	/* DHCP process */
	dhcp.wg.Add(1) // add wainting queue

	dhcp.request = dhcpRequest
	dhcp.child = child
	adapter.DHCPstagePacket(dhcp)
	adapter.DHCPsendStagedPackets()

	dhcp.wg.Wait() // Waiting for DHCP request transaction.

	if dhcp != nil {
		adapter.PutDHCPElement(dhcp)
	}

	child.running.Wait() // Waiting for receiving Registration Response.

	// If isLogin and isVipAssign is true, start the child device.
	// After sending DHCP ACK and receiving Registration Response, start the child device.
	if child.isLogin.Get() && child.isVipAssign.Get() {
		child.Start()
	} else {
		logger.Warn(fmt.Sprintf("%s is unauthenticated or has a closed state", child.deviceName))
	}
}

// RecoverAllChildDevices checks liveness of previously connected devices.
func (adapter *AdapterDevice) RecoverAllChildDevices(nodeIDCache *cache.NodeIDCache, ipVersion string) {
	if adapter.redisClient == nil {
		logger.Warn("Redis client is not initialized, skipping device recovery")
		return
	}

	virtualIPs, err := adapter.redisClient.GetAllVirtualIPs()
	if err != nil {
		logger.Error(fmt.Errorf("failed to get virtual IPs from Redis: %w", err))
		return
	}

	if len(virtualIPs) == 0 {
		logger.Info("No previous devices found in Redis, skipping device recovery")
		return
	}

	var sourceAddr string
	if ipVersion == "4" {
		sourceAddr = adapter.InternalInterface.Addr4.String()
	} else {
		sourceAddr = adapter.InternalInterface.Addr6.String()
	}

	var wg sync.WaitGroup

	for _, vipInfo := range virtualIPs {
		var targetAddr string
		if ipVersion == "4" {
			targetAddr = vipInfo.VirtualIPv4
		} else {
			targetAddr = vipInfo.VirtualIPv6
		}

		if targetAddr == "" {
			logger.Debug(fmt.Sprintf("Skipping %s: no IPv%s address", vipInfo.DeviceName, ipVersion))
			continue
		}

		macAddr, err := net.ParseMAC(vipInfo.MacAddress)
		if err != nil {
			logger.Error(fmt.Errorf("failed to parse MAC address %s: %w", vipInfo.MacAddress, err))
			continue
		}

		// Check if this device is in our children map
		adapter.children.RLock()
		child, exists := adapter.children.keyMap[macAddr.String()]
		adapter.children.RUnlock()

		if !exists {
			logger.Warn(fmt.Sprintf("Device %s (MAC: %s) found in Redis but not in children map, skipping", vipInfo.DeviceName, vipInfo.MacAddress))
			continue
		}

		// Run ping per device in parallel so slow or unreachable devices don't delay overall startup readiness
		wg.Add(1)
		go func(targetAddr, macAddrStr string, child *ChildDevice) {
			defer wg.Done()

			// Ping the device to check if it's still alive
			logger.Debug(fmt.Sprintf("Pinging %s (%s) from %s", vipInfo.DeviceName, targetAddr, sourceAddr))

			stats, err := cmd.ExecutePingCmd(targetAddr, sourceAddr, 3)
			if err != nil {
				logger.Error(fmt.Errorf("failed to ping %s (%s): %w", vipInfo.DeviceName, targetAddr, err))
				return
			}

			if stats.PacketsRecv == 0 {
				logger.Info(fmt.Sprintf("Device %s (%s) did not respond to ping, removing from Redis", vipInfo.DeviceName, targetAddr))

				// Delete the Virtual IP entry from Redis as the device is not reachable
				if err := adapter.redisClient.DeleteVirtualIP(macAddrStr); err != nil {
					logger.Error(fmt.Errorf("failed to delete virtual IP from Redis for %s: %w", vipInfo.DeviceName, err))
				}

				return
			}

			go adapter.RecoverChildDevice(child, nodeIDCache)
		}(targetAddr, macAddr.String(), child)
	}

	wg.Wait()
}

// RecoverChildDevice re-executes authentication and registration for a previously connected device.
func (adapter *AdapterDevice) RecoverChildDevice(child *ChildDevice, nodeIDCache *cache.NodeIDCache) {
	child.Lock()
	defer child.Unlock()

	if child.isLogin.Get() && child.isUp() {
		logger.Info(fmt.Sprintf("Device %s is already logged in and running, skipping recovery", child.deviceName))
		return
	}

	logger.Info(fmt.Sprintf("Starting recovery for device %s", child.deviceName))

	child.state.state = uint32(deviceStateDown)

	connTLS, err := adapter.MakeTLSConn()
	if err != nil {
		logger.Error(fmt.Errorf("failed to establish tls connection for recovery: %w", err))
		return
	}

	authElem, err := layers.SetAuthenticationElement(child.authType, child.deviceID, child.hashedPassword, child.certificatePath)
	if err != nil {
		logger.Error(fmt.Errorf("failed to set authentication element for recovery: %w", err))
		return
	}

	logger.Debug(fmt.Sprintf("Execute authentication process for recovery: %s", string(authElem.DeviceID)))

	lres := HandleAuthentication(connTLS, authElem)
	logger.Debug(fmt.Sprintf("Successfully executed %s's authentication process for recovery", child.deviceName))

	child.fqdn = lres.FQDN
	child.nodeID = lres.BaseHeader.ID
	child.VirtualIPv4 = lres.VirtualIPv4Address
	child.VirtualIPv6 = lres.VirtualIPv6Address
	child.commonKey = lres.CommonKey

	logger.Info(fmt.Sprintf("ChildDevice Recovery - DeviceName=%s", child.deviceName))
	logger.Info(fmt.Sprintf("ChildDevice Recovery - NodeID=%x", lres.BaseHeader.ID))
	logger.Info(fmt.Sprintf("ChildDevice Recovery - VirtualIPv4=%v", lres.VirtualIPv4Address))
	logger.Info(fmt.Sprintf("ChildDevice Recovery - VirtualIPv6=%v", lres.VirtualIPv6Address))

	// Update Redis with latest information
	if adapter.redisClient != nil {
		if err := adapter.redisClient.SaveVirtualIP(
			child.deviceName,
			string(child.deviceID),
			child.MacAddress.String(),
			lres.VirtualIPv4Address,
			lres.VirtualIPv6Address,
		); err != nil {
			logger.Error(fmt.Errorf("failed to update virtual IP information in Redis: %w", err))
		}
	}

	child.isLogin.Set(true)

	if adapter.Cfg.Adapterd.VirtualIPType == "4" {
		child.isVirtualIPv6.Set(false)
	} else {
		child.isVirtualIPv6.Set(true)
	}

	logger.Debug(fmt.Sprintf("Execute registration process for recovery: %x", child.nodeID))

	if err := child.HandleChildDeviceRegistration(nodeIDCache); err != nil {
		logger.Error(fmt.Errorf("failed to execute child device's registration process for recovery: %w", err))
		return
	} else {
		logger.Debug(fmt.Sprintf("Successfully executed %s's registration process for recovery", child.deviceName))
		child.running.Add(1) // add waiting queue
	}

	child.isVipAssign.Set(true)
	child.running.Wait() // Waiting for receiving Registration Response.

	// Start the child device
	if child.isLogin.Get() && child.isVipAssign.Get() {
		child.Start()
		logger.Info(fmt.Sprintf("Successfully recovered device %s", child.deviceName))
	} else {
		logger.Warn(fmt.Sprintf("Failed to recover %s: authentication or IP assignment failed", child.deviceName))
	}
}

// Start starts the child device state.
// When the child device state is up, the general node is ready for CYPHONIC communication.
func (child *ChildDevice) Start() {
	var ctx context.Context

	// If the child device state is already running, the process is skipped.
	// flag: (aiving) flag must be true.
	if child.isDown() || child.isClosed() {
		child.state.state = uint32(deviceStateUp)   // child device state is up.
		child.peers.keyMap = make(map[string]*Peer) // Initialize Peer table.
		child.tunnelCache = NewTunnelInfoCache()    // Initialize tunnelCache.

		ctx, child.routineCancel = context.WithCancel(context.Background())
		go child.RoutineAlivenessConfirmation(ctx)

		logger.Info(fmt.Sprintf("%s is running", child.deviceName))
	} else if child.isUp() {
		logger.Info(fmt.Sprintf("%s is already in running state", child.deviceName))
	} else {
		logger.Error(fmt.Errorf("failed to bring %s to running state", child.deviceName))
	}
}

// Stop closes the state of the child device.
// When the child device state is close, the general node is not ready for CYPHONIC communication.
//
/*** child device stopping conditions ***/
/*
 * - If aliveness cannot be confirmed by RoutineAlivenessConfirmation.
 * - If there is no response when pinging Route Direction to Responder.
 */
func (child *ChildDevice) Stop() {
	child.state.state = uint32(deviceStateClosed) // child device state is close.
	child.peers.keyMap = make(map[string]*Peer)   // Initialize Peer table.
	child.tunnelCache = NewTunnelInfoCache()      // Initialize tunnelCache.

	// Safely stop RoutineAlivenessConfirmation (Graceful shutdown for RoutineAlivenessConfirmation)
	// You can call the Stop() method flexibly.
	// Otherwise, the RoutineAlivenessConfirmation continues to remain.
	child.routineCancel()

	// TODO:
	// The three routines (ChildRoutineSequentialSender, ChildRoutineSequentialReceiver,
	// ChildRoutineSequentialKeepAlivePeer) generated during peer creation will also be stopped.

	logger.Info(fmt.Sprintf("%s is closed", child.deviceName))
}

// RoutineAlivenessConfirmation checks if the general node is running every 3 minutes
// Execute ping 3 times at 1 second intervals, and if all fails, close the child device
// This routing will stop if there is no ping response or if a cancellation signal is received.
func (child *ChildDevice) RoutineAlivenessConfirmation(ctx context.Context) {
	t := time.NewTicker(ChildDeviceLivenessProbeInterval)
	var targetAddr string
	var sourceAddr string

	defer func() {
		t.Stop()
		logger.Debug(fmt.Sprintf("Routine: Routine Aliveness Confirmation - stopped: [child name=%s]", child.deviceName))
		child.Stop()
	}()
	logger.Debug(fmt.Sprintf("Routine: Routine Aliveness Confirmation - started: [child name=%s]", child.deviceName))

	targetAddr = child.VirtualIPv4.String()
	sourceAddr = child.Adapter.InternalInterface.Addr4.String()

	// child device as IPv6 mode configured.
	if child.isVirtualIPv6.Get() {
		targetAddr = child.VirtualIPv6.String()
		sourceAddr = child.Adapter.InternalInterface.Addr6.String()
	}

	for {
		select {
		case <-t.C:
			stats, err := cmd.ExecutePingCmd(targetAddr, sourceAddr, ChildDeviceLivenessProbeCount)
			if err != nil {
				logger.Error(fmt.Errorf("Routine: Routine Aliveness Confirmation error: %w", err))
				return
			}

			if stats.PacketsRecv == 0 {
				logger.Debug(fmt.Sprintf("%s is dead", child.deviceName))
				return
			}

			logger.Debug(fmt.Sprintf("%s is aliving", child.deviceName))
		case <-ctx.Done():
			logger.Debug("Cancellation signal received")
			return
		}
	}
}

// ReadChildDeviceInformation reads ChildDevice information from a JSON file.
func (adapter *AdapterDevice) ReadChildDeviceInformation() error {
	var infos []layers.AcquisitionResponse
	var info layers.AcquisitionResponse
	var index int

	b, err := os.ReadFile(adapter.Cfg.Adapterd.ChildDeviceInformationPath)
	if err != nil {
		return fmt.Errorf("failed to get child device information path: %w", err)
	}

	if err := json.Unmarshal(b, &infos); err != nil {
		return fmt.Errorf("failed to json unmarshal Acquisition Response: %w", err)
	}

	for index, info = range infos {
		adapter.child, err = adapter.SetChildDeviceInformation(info)
		if err != nil {
			return fmt.Errorf("failed to set child device information: %w", err)
		}

		logger.Info(fmt.Sprintf("GeneralNode Information=%s", adapter.child.deviceName))

		adapter.child.Adapter = adapter
	}

	logger.Info(fmt.Sprintf("child device information count=%d", index+1))

	return nil
}

// SetChildDeviceInformation sets ChildDevice information.
func (adapter *AdapterDevice) SetChildDeviceInformation(info layers.AcquisitionResponse) (*ChildDevice, error) {
	var err error

	child := new(ChildDevice)
	child.Lock()
	defer child.Unlock()

	// DeviceName
	child.deviceName = info.DeviceName

	// DeviceID
	child.deviceID = []byte(info.DeviceID)

	// Password
	password, err := hex.DecodeString(info.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to convert strig to byte in child device's Password: %w", err)
	}
	copy(child.hashedPassword[:], password)

	// AuthType
	child.authType = info.AuthType

	// MacAddress
	child.MacAddress, err = net.ParseMAC(info.MacAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MAC address: %w", err)
	}

	// EnableIPv6
	child.isVirtualIPv6.Set(info.EnableIPv6)

	_, ok := adapter.children.keyMap[child.MacAddress.String()]
	if ok {
		return nil, errors.New("child device already exists")
	}
	adapter.children.keyMap[child.MacAddress.String()] = child

	return child, nil
}
