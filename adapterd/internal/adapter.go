package internal

import (
	"fmt"
	"net"
	"net/netip"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/config"
	"github.com/Pluslab/cyphonic-adapter/adapterd/domain"
	"github.com/Pluslab/cyphonic-adapter/adapterd/internal/cache"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ndp"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

const (
	keepAliveToNMSInterval         = 5 * time.Second
	keepAliveToChildDeviceInterval = 20 * time.Second
)

// adapterState returns device.state.state as a deviceState
// See those docs for how to interpret this value.
func (adapter *AdapterDevice) adapterState() deviceState {
	return deviceState(atomic.LoadUint32(&adapter.state.state))
}

// isDown reports whether the adapter device is not configured (or is attempting to come up).
// See device.state.state comments for how to interpret this value.
func (adapter *AdapterDevice) isDown() bool {
	return adapter.adapterState() == deviceStateDown
}

// isUp reports whether the adapter device is up (or is attempting to come up).
// See device.state.state comments for how to interpret this value.
func (adapter *AdapterDevice) isUp() bool {
	return adapter.adapterState() == deviceStateUp
}

// isClosed reports whether the adapter device is closed (or is closing).
// See device.state.state comments for how to interpret this value.
func (adapter *AdapterDevice) isClosed() bool {
	return adapter.adapterState() == deviceStateClosed
}

// AdapterDevice manages pooling, queues.
// Pool manages packet buffers.
// Queue manages inbound, outbound queue to encrypt and decrypt.
type AdapterDevice struct {
	Cfg *config.Config

	redisClient *Client

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

	ConnUDP      *net.UDPConn
	childConnUDP *net.UDPConn
	NMSAddrv4    net.UDPAddr
	NMSAddrv6    net.UDPAddr
	commonKey    []byte

	// InternalIPVersion is the ChildDevice's working IP version (Virtual IP version).
	// ExternalIPVersion is the adapter's working IP version (Real IP version).
	InternalIPVersion layers.TypeLocalIPVersion
	ExternalIPVersion layers.TypeLocalIPVersion

	nodeID []byte
	fqdn   []byte

	pool struct {
		messageBuffers   *WaitPool
		inboundElements  *WaitPool
		outboundElements *WaitPool
		arpElements      *WaitPool
		dnsElements      *WaitPool
		dhcpElements     *WaitPool
		ndpElements      *WaitPool
	}

	queue struct {
		encryption *outboundQueue
		decryption *inboundQueue
	}

	adapterQueue struct {
		arpStaged  chan *QueueARPElement
		arp        *autodrainingARPQueue
		dnsStaged  chan *QueueDNSElement
		dns        *autodrainingDNSQueue
		dhcpStaged chan *QueueDHCPElement
		dhcp       *autodrainingDHCPQueue
		ndpStaged  chan *QueueNDPElement
		ndp        *autodrainingNDPQueue
	}

	InternalInterface struct {
		Card          *net.Interface
		Addr4         netip.Addr
		Addr6         netip.Addr
		LinkLocalAddr netip.Addr
	}

	socket struct {
		receiveIPv4 int // receive IPv4 socket
		receiveIPv6 int // receive IPv6 socket
		sendIPv4    int // send IPv4 socket
		sendIPv6    int // send IPv6 socket
	}

	// Manages general node information in an internal cache.
	// In a closed network configured at the link layer level,
	// devices are usually identified by their MAC addresses.
	children struct {
		sync.RWMutex
		keyMap map[string]*ChildDevice // - key: MacAddress / value: *ChildDevice
	}

	child *ChildDevice
}

func (adapter *AdapterDevice) CloseAdapterDevice() error {
	adapter.state.stopping.Wait()      // Waiting for all routines to complete.
	adapter.queue.encryption.wg.Wait() // Waiting for all routines to complete.

	// Executing after all routines are safely stopped.
	if err := adapter.ConnUDP.Close(); err != nil {
		return fmt.Errorf("failed to close udp connection: %w", err)
	}

	return nil
}

func (adapter *AdapterDevice) SetInternalIPVersion() {
	cfg, err := config.Get()
	if err != nil {
		logger.Error(fmt.Errorf("failed to get config: %w", err))
	}

	if cfg.Adapterd.VirtualIPType == "4" {
		adapter.InternalIPVersion = layers.TypeLocalIPVersion4
	} else if cfg.Adapterd.VirtualIPType == "6" {
		adapter.InternalIPVersion = layers.TypeLocalIPVersion6
	} else {
		logger.Error(fmt.Errorf("unsupported virtual IP type: %s", cfg.Adapterd.VirtualIPType))
	}
}

// NewAdapter initializes the adapter device.
func (adapter *AdapterDevice) NewAdapterDevice(dnsCh chan DNSQueue, dnsCache *cache.DNSCache, dhcpCh chan DHCPQueue) error {
	var err error
	adapter.state.state = uint32(deviceStateDown)
	adapter.SetInternalIPVersion()

	authType := adapter.Cfg.Adapterd.AuthType
	adapterID := []byte(adapter.Cfg.Adapterd.AdapterID)
	hashedPassword := layers.GenerateHashedPassword([]byte(adapter.Cfg.Adapterd.Password))
	certificatePath := adapter.Cfg.Adapterd.LoginRequestCertificatePath

	authElem, err := layers.SetAuthenticationElement(authType, adapterID, hashedPassword, certificatePath)
	if err != nil {
		return fmt.Errorf("failed to set authentication element: %w", err)
	}

	// UDP connection
	adapter.ConnUDP, err = MakeUDPConn()
	if err != nil {
		return fmt.Errorf("failed to establish udp connection: %w", err)
	}

	// TLS connection
	connTLS, err := adapter.MakeTLSConn()
	if err != nil {
		return fmt.Errorf("failed to establish tls connection: %w", err)
	}

	// HTTP client
	httpClient, err := adapter.MakeHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create http client: %w", err)
	}

	/* Authentication process */
	logger.Debug(fmt.Sprintf("Execute authentication process=%s", string(adapterID)))

	lres := HandleAuthentication(connTLS, authElem)
	logger.Debug("Successfully executed authentication process")

	adapter.nodeID = lres.BaseHeader.ID // adapter's nodeID
	adapter.fqdn = lres.FQDN            // adapter's FQDN
	adapter.commonKey = lres.CommonKey  // adapter - NMS common key

	logger.Info(fmt.Sprintf("Adapter's ID (NodeID)=%x", lres.BaseHeader.ID))
	logger.Info(fmt.Sprintf("Adapter's FQDN=%s", string(lres.FQDN)))
	logger.Info(fmt.Sprintf("Adapter's VirtualIPv4=%v", lres.VirtualIPv4Address))
	logger.Info(fmt.Sprintf("Adapter's VirtualIPv6=%v", lres.VirtualIPv6Address))

	adapter.NMSAddrv4 = net.UDPAddr{
		IP:   net.IP(lres.NMSIPv4Address.AsSlice()).To4(),
		Port: adapter.Cfg.Global.NMSPort,
	}

	adapter.NMSAddrv6 = net.UDPAddr{
		IP:   net.IP(lres.NMSIPv6Address.AsSlice()).To16(),
		Port: adapter.Cfg.Global.NMSPort,
	}

	/* Registration process */
	logger.Debug(fmt.Sprintf("Execute registration process=%x", adapter.nodeID))

	if err = adapter.HandleRegistration(); err != nil {
		return fmt.Errorf("failed to execute registration process: %w", err)
	} else {
		logger.Debug("Successfully executed registration process")
	}

	adapter.children.keyMap = make(map[string]*ChildDevice)

	/* Acquisition process */
	logger.Debug(fmt.Sprintf("Execute acquisition process=%s", adapter.Cfg.Adapterd.AdapterID))

	if err = adapter.HandleAcquisition(httpClient); err != nil {
		return fmt.Errorf("failed to execute acquisition process: %w", err)
	} else {
		logger.Debug("Successfully executed acquisition process")
	}

	adapter.redisClient, err = NewClient(adapter.Cfg)
	if err != nil {
		adapter.redisClient = nil

		logger.Warn(fmt.Sprintf("failed to connect to Redis: %v", err))
	}

	// Set adapter internal interface
	if err := adapter.SetInternalInterface(); err != nil {
		return fmt.Errorf("failed to set adapter internal interface: %w", err)
	}

	adapterDevice := adapter.InternalInterface.Card

	// create raw socket descriptor
	adapter.CreateDescriptor(adapterDevice)

	logger.Info(fmt.Sprintf("Adapter's Internal Interface Addr4=%v", adapter.InternalInterface.Addr4))
	logger.Info(fmt.Sprintf("Adapter's Internal Interface Addr6=%v", adapter.InternalInterface.Addr6))
	logger.Info(fmt.Sprintf("Adapter's Internal Interface LinkLocalAddr=%v", adapter.InternalInterface.LinkLocalAddr))
	logger.Info(fmt.Sprintf("Adapter's Internal Interface HardwareAddr=%v", adapter.InternalInterface.Card.HardwareAddr))
	domain.ProvisionedMessage()

	adapter.state.state = uint32(deviceStateUp) // adapter device state is up

	// create pools
	adapter.PopulatePools()

	// create queues
	adapter.queue.encryption = newOutboundQueue() // Outgoing packet
	adapter.queue.decryption = newInboundQueue()  // Incoming packet

	adapter.adapterQueue.arp = newAutodrainingARPQueue(adapter) // ARP packet
	adapter.adapterQueue.arpStaged = make(chan *QueueARPElement, QueueStagedSize)

	adapter.adapterQueue.dns = newAutodrainingDNSQueue(adapter) // DNS packet
	adapter.adapterQueue.dnsStaged = make(chan *QueueDNSElement, QueueStagedSize)

	adapter.adapterQueue.dhcp = newAutodrainingDHCPQueue(adapter) // DHCP packet
	adapter.adapterQueue.dhcpStaged = make(chan *QueueDHCPElement, QueueStagedSize)

	adapter.adapterQueue.ndp = newAutodrainingNDPQueue(adapter) // NDP packet
	adapter.adapterQueue.ndpStaged = make(chan *QueueNDPElement, QueueStagedSize)

	// start workers
	cpus := runtime.NumCPU()

	adapter.queue.encryption.wg.Add(cpus) // One for each RoutineHandshake

	for i := 0; i < cpus; i++ {
		go adapter.RoutineEncryption()
		go adapter.RoutineDecryption()
	}

	adapter.state.stopping.Add(7)      // RoutineKeepAliveNMS, RoutineKeepAliveGeneralDevices, RoutineSendOutgoing, RoutineDHCPv4 or RoutineDHCPv6, RoutineAnalysisDNS, RoutineARP
	adapter.queue.encryption.wg.Add(1) // RoutineSendOutgoing

	go adapter.RoutineKeepAliveNMS()
	go adapter.RoutineSendOutgoing()
	go adapter.RoutineAnalysisDNS(dnsCh, dnsCache)
	go adapter.RoutineARP()
	go adapter.RoutineNDP()

	switch adapter.InternalIPVersion {
	case layers.TypeLocalIPVersion4:
		go adapter.RoutineDHCPv4()
		adapter.AdapterDHCPListen(dhcpCh) // Default: listen port: 67 (DHCP)

	case layers.TypeLocalIPVersion6:
		go adapter.RoutineDHCPv6()
		go adapter.RoutineKeepAliveGeneralDevices()
		adapter.AdapterDHCPv6Listen(dhcpCh) // Default: listen port: 547 (DHCPv6)

	default:
		return fmt.Errorf("unsupported internal IP version: %v", adapter.InternalIPVersion)
	}

	return nil
}

func (adapter *AdapterDevice) RoutineKeepAliveGeneralDevices() {
	t := time.NewTicker(keepAliveToNMSInterval)
	defer func() {
		t.Stop()
		adapter.state.stopping.Done()
		logger.Debug("Routine: KeepAlive to General Devices - stopped")
	}()
	logger.Debug("Routine: KeepAlive to General Devices - started")

	for {
		<-t.C
		if err := adapter.KeepAliveforGeneralDevices(); err != nil {
			logger.Error(fmt.Errorf("failed to execute keep aliving to General Devices: %w", err))
		}
	}
}

func (adapter *AdapterDevice) KeepAliveforGeneralDevices() error {
	srcAddr, err := netip.ParseAddr(ip.AllNodeMulticastAddress)
	if err != nil {
		return fmt.Errorf("failed to parse address: %w", err)
	}

	if ra, err := ndp.GenerateRouterAdvertise(
		adapter.InternalInterface.LinkLocalAddr,
		adapter.InternalInterface.Card.HardwareAddr,
		srcAddr,
	); err != nil {
		logger.Error(fmt.Errorf("failed to send Keep Alive packet to General Devices: %w", err))
	} else {
		if err := SendEtherPacket(adapter.socket.receiveIPv6, ra); err != nil {
			logger.Error(fmt.Errorf("failed to send Keep Alive packet to General Devices: %w", err))
		} else {
			logger.Info(fmt.Sprintf("Send Keep Alive to %v (General Devices)", srcAddr))
		}
	}
	return nil
}

// KeepAlivePeer executes keep alive to Node Management Service.
func (adapter *AdapterDevice) RoutineKeepAliveNMS() {
	t := time.NewTicker(keepAliveToNMSInterval)
	defer func() {
		t.Stop()
		adapter.state.stopping.Done()
		logger.Debug("Routine: KeepAlive to NMS - stopped")
	}()
	logger.Debug("Routine: KeepAlive to NMS - started")

	for {
		<-t.C
		if err := adapter.KeepAlive(); err != nil {
			logger.Error(fmt.Errorf("failed to execute keep aliving to NMS: %w", err))
		}
	}
}

// KeepAlive sends keep alive packet to Node Management Service.
// The adapter device performs keep alive on behalf of child device.
func (adapter *AdapterDevice) KeepAlive() error {
	// https://cyphonic.esa.io/posts/45#%E3%80%90%E9%87%8D%E8%A6%81%E3%80%91BeaseHeader.ID
	kp := layers.GenerateKeepAlive(adapter.nodeID)

	packet, err := kp.Marshal()
	if err != nil {
		logger.Error(fmt.Errorf("failed to marshal Keep Alive packet: %w", err))
	}

	switch adapter.ExternalIPVersion {
	case layers.TypeLocalIPVersion4:
		_, err := adapter.ConnUDP.WriteToUDPAddrPort(packet, adapter.NMSAddrv4.AddrPort())
		if err != nil {
			return fmt.Errorf("failed to send Keep Alive packet to %v: %w", adapter.NMSAddrv4.AddrPort(), err)
		} else {
			logger.Debug(fmt.Sprintf("Send Keep Alive to %v (NMS)", adapter.NMSAddrv4.AddrPort()))
		}

	case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
		_, err := adapter.ConnUDP.WriteToUDPAddrPort(packet, adapter.NMSAddrv6.AddrPort())
		if err != nil {
			return fmt.Errorf("failed to send Keep Alive packet to %v: %w", adapter.NMSAddrv6.AddrPort(), err)
		} else {
			logger.Debug(fmt.Sprintf("Send Keep Alive to %v (NMS)", adapter.NMSAddrv6.AddrPort()))
		}
	}

	return nil
}
