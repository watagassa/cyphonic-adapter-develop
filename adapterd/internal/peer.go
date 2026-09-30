package internal

import (
	"container/list"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// QueueStagedSize is staged buffer size.
const QueueStagedSize = 128

const endpointLocalPort = 30000

const keepAliveToPeerInterval = 5 * time.Second

// tunnelPendingTimeout is the standard DNS name resolution timeout.
const tunnelPendingTimeout = 1 * time.Second

type TunnelState uint8

const (
	TunnelStateNone TunnelState = iota
	TunnelStatePending
	TunnelStateEstablished
)

// Peer is to manage peer information.
type Peer struct {
	sync.RWMutex
	child          *ChildDevice
	connUDP        *net.UDPConn
	endpointv4     net.UDPAddr
	endpointv6     net.UDPAddr
	endpointTRSv4  net.UDPAddr
	endpointTRSv6  net.UDPAddr
	tunnelKey      []byte
	tempKey        []byte
	endKey         []byte
	pathID         []byte
	dstVirtualIPv4 netip.Addr
	dstVirtualIPv6 netip.Addr
	nodeType       AtomicBool     // True is Mobile Node, False is Correspondent Node
	stopping       sync.WaitGroup // routines pending stop
	isRunning      AtomicBool
	isTRS          AtomicBool

	state struct {
		sync.Mutex // protects against concurrent Start/Stop
	}

	queue struct {
		staged   chan *QueueOutboundElement // staged packets before a handshake is available
		outbound *autodrainingOutboundQueue // sequential ordering of udp transmission
		inbound  *autodrainingInboundQueue  // sequential ordering of tun writing
	}
	trieEntries list.List
}

// Tunnel Information Cache
// Key: PeerFQDN (Responder FQDN) / Value: Peer (Tunnel Table)
//
// Put: When creating a peer.
// Get: When receiving a DNS request.
// Update: Nothing.
// Expire: When deleting Peer.
type tunnelInfo struct {
	peer    *Peer // peer: Tunnel information
	state   TunnelState
	expires int64
}

// TunnelInfoCache is a structure for managing the cache of tunnel information.
type TunnelInfoCache struct {
	tunnelInfos map[string]*tunnelInfo
	mu          sync.Mutex
}

// NewTunnelInfoCache makes tunnel information cache available.
func NewTunnelInfoCache() *TunnelInfoCache {
	c := &TunnelInfoCache{tunnelInfos: make(map[string]*tunnelInfo)}
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			<-t.C
			c.mu.Lock()
			for k, v := range c.tunnelInfos {
				if v.state == TunnelStatePending && time.Now().UnixNano() > v.expires {
					delete(c.tunnelInfos, k)
					logger.Warn(fmt.Sprintf("Pending tunnel entry expired: dstFQDN=%s", k))
				}
			}
			c.mu.Unlock()
		}
	}()

	return c
}

// PutPending registers a pending tunnel entry for the given dstFQDN.
func (c *TunnelInfoCache) PutPending(dstFQDN string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.tunnelInfos[dstFQDN]; !ok {
		c.tunnelInfos[dstFQDN] = &tunnelInfo{
			peer:    nil,
			state:   TunnelStatePending,
			expires: time.Now().Add(tunnelPendingTimeout).UnixNano(),
		}
	}
}

// PutEstablished registers an established tunnel entry for the given dstFQDN and Peer.
func (c *TunnelInfoCache) PutEstablished(dstFQDN string, peer *Peer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tunnelInfos[dstFQDN] = &tunnelInfo{
		peer:  peer,
		state: TunnelStateEstablished,
	}
}

// Get gets peer (tunnel information) and tunnel state for the given dstFQDN.
func (c *TunnelInfoCache) Get(dstFQDN string) (*Peer, TunnelState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.tunnelInfos[dstFQDN]; ok {
		return v.peer, v.state
	}

	return nil, TunnelStateNone
}

// NewPeer init new peer of a general node.
func (child *ChildDevice) NewPeer(rd *layers.RouteDirection, nodeType nodeType, endKey []byte) (*Peer, error) {
	if child.isClosed() {
		err := errors.New("child device closed")
		return nil, err
	}

	child.peers.Lock()
	defer child.peers.Unlock()

	// create peer
	peer := new(Peer)
	peer.Lock()
	defer peer.Unlock()

	tunnelKey := make([]byte, 16)
	tempKey := make([]byte, 16)
	end := make([]byte, 16)
	pathID := make([]byte, 16)
	copy(tunnelKey, rd.TunnelKey)
	copy(tempKey, rd.TemporaryKey)
	copy(end, endKey)
	copy(pathID, rd.PathID)

	peer.child = child                   // ChildDevice information
	peer.connUDP = child.Adapter.ConnUDP // UDP connection
	peer.tunnelKey = tunnelKey           // Tunnel key
	peer.tempKey = tempKey               // Temp key
	peer.endKey = end                    // End key
	peer.pathID = pathID                 // PathID

	trsipv4 := rd.TRSIPv4 // TRS IPv4 address
	trsipv6 := rd.TRSIPv6 // TRS IPv6 address

	switch nodeType {
	case nodeTypeMN:
		endipv4 := rd.NATcnIPv4
		endipv6 := rd.NATcnIPv6
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(endipv4.AsSlice()).To4(),
			Port: int(rd.NATcnPort),
		}
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(endipv6.AsSlice()).To16(),
			Port: int(rd.NATcnPort),
		}
		peer.endpointTRSv4 = net.UDPAddr{
			IP:   net.IP(trsipv4.AsSlice()).To4(),
			Port: child.Adapter.Cfg.Global.TRSPort,
		}
		peer.endpointTRSv6 = net.UDPAddr{
			IP:   net.IP(trsipv6.AsSlice()).To16(),
			Port: child.Adapter.Cfg.Global.TRSPort,
		}
		peer.nodeType.Set(true)
	case nodeTypeCN:
		endipv4 := rd.NATmnIPv4
		endipv6 := rd.NATmnIPv6
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(endipv4.AsSlice()).To4(),
			Port: int(rd.NATmnPort),
		}
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(endipv6.AsSlice()).To16(),
			Port: int(rd.NATmnPort),
		}
		peer.endpointTRSv4 = net.UDPAddr{
			IP:   net.IP(trsipv4.AsSlice()).To4(),
			Port: child.Adapter.Cfg.Global.TRSPort,
		}
		peer.endpointTRSv6 = net.UDPAddr{
			IP:   net.IP(trsipv6.AsSlice()).To16(),
			Port: child.Adapter.Cfg.Global.TRSPort,
		}
		peer.nodeType.Set(false)
	case nodeTypeLocalMN:
		endipv4 := rd.CNRealIPv4
		endipv6 := rd.CNRealIPv6
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(endipv4.AsSlice()).To4(),
			Port: endpointLocalPort,
		}
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(endipv6.AsSlice()).To16(),
			Port: endpointLocalPort,
		}
		peer.endpointTRSv4 = net.UDPAddr{
			IP:   net.IP(trsipv4.AsSlice()).To4(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.endpointTRSv6 = net.UDPAddr{
			IP:   net.IP(trsipv6.AsSlice()).To16(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.nodeType.Set(true)
	case nodeTypeLocalCN:
		endipv4 := rd.MNRealIPv4
		endipv6 := rd.MNRealIPv6
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(endipv4.AsSlice()).To4(),
			Port: endpointLocalPort,
		}
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(endipv6.AsSlice()).To16(),
			Port: endpointLocalPort,
		}
		peer.endpointTRSv4 = net.UDPAddr{
			IP:   net.IP(trsipv4.AsSlice()).To4(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.endpointTRSv6 = net.UDPAddr{
			IP:   net.IP(trsipv6.AsSlice()).To16(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.nodeType.Set(false)
	}

	peer.queue.outbound = newAutodrainingOutboundQueue(child.Adapter)
	peer.queue.inbound = newAutodrainingInboundQueue(child.Adapter)
	peer.queue.staged = make(chan *QueueOutboundElement, QueueStagedSize)

	if rd.ProcessCode == layers.TunnelRequestToTRS {
		peer.isTRS.Set(true)
	} else {
		peer.isTRS.Set(false)
	}

	_, ok := child.peers.keyMap[hex.EncodeToString(rd.PathID)]
	if ok {
		err := errors.New("adding existing peer")
		return nil, err
	}

	// Add to child device's peer table.
	child.peers.keyMap[hex.EncodeToString(rd.PathID)] = peer
	if child.isUp() {
		if err := peer.Start(); err != nil {
			return nil, fmt.Errorf("failed to start peer: %w", err)
		}
	}

	// Add to child device's tunnel information cache.
	child.tunnelCache.PutEstablished(string(rd.CNFQDN), peer)

	switch child.isVirtualIPv6.Get() {
	case false:
		switch nodeType {
		case nodeTypeMN, nodeTypeLocalMN:
			peer.dstVirtualIPv4 = rd.CNVirtualIPv4
			child.allowedips.Insert(netip.PrefixFrom(peer.dstVirtualIPv4, ip.VIPv4PrefixLen), peer)
		case nodeTypeCN, nodeTypeLocalCN:
			peer.dstVirtualIPv4 = rd.MNVirtualIPv4
			child.allowedips.Insert(netip.PrefixFrom(peer.dstVirtualIPv4, ip.VIPv4PrefixLen), peer)
		}
	case true:
		switch nodeType {
		case nodeTypeMN, nodeTypeLocalMN:
			peer.dstVirtualIPv6 = rd.CNVirtualIPv6
			child.allowedips.Insert(netip.PrefixFrom(peer.dstVirtualIPv6, ip.VIPv6PrefixLen), peer)
		case nodeTypeCN, nodeTypeLocalCN:
			peer.dstVirtualIPv6 = rd.MNVirtualIPv6
			child.allowedips.Insert(netip.PrefixFrom(peer.dstVirtualIPv6, ip.VIPv6PrefixLen), peer)
		}
	}

	return peer, nil
}

// Start starts peer state.
func (peer *Peer) Start() error {
	// should never start a peer on a closed child device
	if peer.child.isClosed() {
		return fmt.Errorf("%s is closed", peer.child.deviceName)
	}

	// prevent simultaneous start/stop operations
	peer.state.Lock()
	defer peer.state.Unlock()

	if !peer.isRunning.Get() {
		child := peer.child

		// reset routine state
		peer.stopping.Wait()
		peer.stopping.Add(3) // ChildRoutineSequentialSender, ChildRoutineSequentialReceiver, ChildRoutineSequentialKeepAlivePeer

		child.Adapter.flushInboundQueue(peer.queue.inbound)
		child.Adapter.flushOutboundQueue(peer.queue.outbound)

		go peer.ChildRoutineSequentialSender()
		go peer.ChildRoutineSequentialReceiver()
		go peer.ChildRoutineSequentialKeepAlivePeer()

		peer.isRunning.Set(true)
	}

	return nil
}

// Update updates peer information.
func (peer *Peer) Update(rd *layers.RouteDirection, nodeType nodeType, endKey []byte) {
	peer.Lock()
	defer peer.Unlock()

	tunnelKey := make([]byte, 16)
	tempKey := make([]byte, 16)
	end := make([]byte, 16)
	pathID := make([]byte, 16)
	copy(tunnelKey, rd.TunnelKey)
	copy(tempKey, rd.TemporaryKey)
	copy(end, endKey)
	copy(pathID, rd.PathID)

	peer.tunnelKey = tunnelKey
	peer.tempKey = tempKey
	peer.endKey = end
	peer.pathID = pathID

	trsipv4 := rd.TRSIPv4
	trsipv6 := rd.TRSIPv6

	switch nodeType {
	case nodeTypeMN:
		endipv4 := rd.NATcnIPv4
		endipv6 := rd.NATcnIPv6
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(endipv4.AsSlice()).To4(),
			Port: int(rd.NATcnPort),
		}
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(endipv6.AsSlice()).To16(),
			Port: int(rd.NATcnPort),
		}
		peer.endpointTRSv4 = net.UDPAddr{
			IP:   net.IP(trsipv4.AsSlice()).To4(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.endpointTRSv6 = net.UDPAddr{
			IP:   net.IP(trsipv6.AsSlice()).To16(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.nodeType.Set(true)
	case nodeTypeCN:
		endipv4 := rd.NATmnIPv4
		endipv6 := rd.NATmnIPv6
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(endipv4.AsSlice()).To4(),
			Port: int(rd.NATmnPort),
		}
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(endipv6.AsSlice()).To16(),
			Port: int(rd.NATmnPort),
		}
		peer.endpointTRSv4 = net.UDPAddr{
			IP:   net.IP(trsipv4.AsSlice()).To4(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.endpointTRSv6 = net.UDPAddr{
			IP:   net.IP(trsipv6.AsSlice()).To16(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.nodeType.Set(false)
	case nodeTypeLocalMN:
		endipv4 := rd.CNRealIPv4
		endipv6 := rd.CNRealIPv6
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(endipv4.AsSlice()).To4(),
			Port: endpointLocalPort,
		}
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(endipv6.AsSlice()).To16(),
			Port: endpointLocalPort,
		}
		peer.endpointTRSv4 = net.UDPAddr{
			IP:   net.IP(trsipv4.AsSlice()).To4(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.endpointTRSv6 = net.UDPAddr{
			IP:   net.IP(trsipv6.AsSlice()).To16(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.nodeType.Set(true)
	case nodeTypeLocalCN:
		endipv4 := rd.MNRealIPv4
		endipv6 := rd.MNRealIPv6
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(endipv4.AsSlice()).To4(),
			Port: endpointLocalPort,
		}
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(endipv6.AsSlice()).To16(),
			Port: endpointLocalPort,
		}
		peer.endpointTRSv4 = net.UDPAddr{
			IP:   net.IP(trsipv4.AsSlice()).To4(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.endpointTRSv6 = net.UDPAddr{
			IP:   net.IP(trsipv6.AsSlice()).To16(),
			Port: peer.child.Adapter.Cfg.Global.TRSPort,
		}
		peer.nodeType.Set(false)
	}

	if rd.ProcessCode == layers.TunnelRequestToTRS {
		peer.isTRS.Set(true)
	} else {
		peer.isTRS.Set(false)
	}

	switch peer.child.isVirtualIPv6.Get() {
	case false:
		switch nodeType {
		case nodeTypeMN, nodeTypeLocalMN:
			peer.dstVirtualIPv4 = rd.CNVirtualIPv4
		case nodeTypeCN, nodeTypeLocalCN:
			peer.dstVirtualIPv4 = rd.MNVirtualIPv4
		}
	case true:
		switch nodeType {
		case nodeTypeMN, nodeTypeLocalMN:
			peer.dstVirtualIPv6 = rd.CNVirtualIPv6
		case nodeTypeCN, nodeTypeLocalCN:
			peer.dstVirtualIPv6 = rd.MNVirtualIPv6
		}
	}
}

// UpdateEndPointAddr updates peer's endpoint address information.
func (peer *Peer) UpdateEndPointAddr(addr netip.AddrPort) {
	peer.Lock()
	defer peer.Unlock()

	adapter := peer.child.Adapter

	switch adapter.ExternalIPVersion {
	case layers.TypeLocalIPVersion4:
		peer.endpointv4 = net.UDPAddr{
			IP:   net.IP(addr.Addr().AsSlice()).To4(),
			Port: int(addr.Port()),
		}

	case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
		peer.endpointv6 = net.UDPAddr{
			IP:   net.IP(addr.Addr().AsSlice()).To16(),
			Port: int(addr.Port()),
		}
	}
}

// Optimize optimizes the sender.
func (peer *Peer) Optimize() {
	peer.Lock()
	defer peer.Unlock()

	peer.isTRS.Set(false)
}

// ChildRoutineSequentialKeepAlivePeer executes keep alive to peer node.
func (peer *Peer) ChildRoutineSequentialKeepAlivePeer() {
	t := time.NewTicker(keepAliveToPeerInterval)
	defer func() {
		t.Stop()
		peer.stopping.Done()
		logger.Debug(fmt.Sprintf("Routine: KeepAlive to Peer - stopped: [child name=%s] [peer vip4=%v] [peer vip6=%v]", peer.child.deviceName, peer.dstVirtualIPv4, peer.dstVirtualIPv6))
	}()
	logger.Debug(fmt.Sprintf("Routine: KeepAlive to Peer - started: [child name=%s]", peer.child.deviceName))

	for {
		<-t.C
		if err := peer.KeepAlive(); err != nil {
			logger.Error(fmt.Errorf("failed to execute keep aliving to peer node: %w", err))
		}
	}
}

// KeepAlive sends keep alive packet to peer node.
// The adapter device performs keep alive on behalf of child device.
func (peer *Peer) KeepAlive() error {
	peer.Lock()
	defer peer.Unlock()

	adapter := peer.child.Adapter

	// https://cyphonic.esa.io/posts/45#%E3%80%90%E9%87%8D%E8%A6%81%E3%80%91BeaseHeader.ID
	kp := layers.GenerateKeepAlive(peer.pathID)

	packet, err := kp.Marshal()
	if err != nil {
		logger.Error(fmt.Errorf("failed to marshal Keep Alive packet: %w", err))
	}

	if peer.isTRS.Get() {
		switch adapter.ExternalIPVersion {
		case layers.TypeLocalIPVersion4:
			_, err := peer.connUDP.WriteToUDPAddrPort(packet, peer.endpointTRSv4.AddrPort())
			if err != nil {
				return fmt.Errorf("failed to send Keep Alive packet to %v: %w", peer.endpointTRSv4.AddrPort(), err)
			} else {
				logger.Debug(fmt.Sprintf("Send Keep Alive to %v (TRS)", peer.endpointTRSv4.AddrPort()))
			}
		case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
			_, err := peer.connUDP.WriteToUDPAddrPort(packet, peer.endpointTRSv6.AddrPort())
			if err != nil {
				return fmt.Errorf("failed to send Keep Alive packet to %v: %w", peer.endpointTRSv6.AddrPort(), err)
			} else {
				logger.Debug(fmt.Sprintf("Send Keep Alive to %v (TRS)", peer.endpointTRSv6.AddrPort()))
			}
		}
	} else {
		switch adapter.ExternalIPVersion {
		case layers.TypeLocalIPVersion4:
			_, err := peer.connUDP.WriteToUDPAddrPort(packet, peer.endpointv4.AddrPort())
			if err != nil {
				return fmt.Errorf("failed to send Keep Alive packet to %v: %w", peer.endpointv4.AddrPort(), err)
			} else {
				logger.Debug(fmt.Sprintf("Send Keep Alive to %v (Peer)", peer.endpointv4.AddrPort()))
			}
		case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
			_, err := peer.connUDP.WriteToUDPAddrPort(packet, peer.endpointv6.AddrPort())
			if err != nil {
				return fmt.Errorf("failed to send Keep Alive packet to %v: %w", peer.endpointv6.AddrPort(), err)
			} else {
				logger.Debug(fmt.Sprintf("Send Keep Alive to %v (Peer)", peer.endpointv6.AddrPort()))
			}
		}
	}

	return nil
}
