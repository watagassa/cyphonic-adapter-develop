package internal

import (
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"net"
	"net/netip"
	"sync"
	"unsafe"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// parentIndirection is an indirect entry table that manages already registered entry tables.
type parentIndirection struct {
	parentBit     **trieEntry
	parentBitType uint8
}

// trieEntry is an entry table.
// It stores the virtual IP addresses of peer nodes and acts as a staging table
// that compares already registered virtual IP addresses with newly added virtual
// IP addresses and decides whether to register them in the alloweds IP table.
type trieEntry struct {
	peer        *Peer
	child       [2]*trieEntry
	parent      parentIndirection // entry table already stored
	cidr        uint8             // peer virtual IP address CIDR
	bitAtByte   uint8
	bitAtShift  uint8
	bits        []byte
	perPeerElem *list.Element
}

// commonBits finds the common bit length from the two IP addresses.
// Typically, this is the prefix (subnetmasks): here cidr.
func commonBits(ip1, ip2 []byte) uint8 {
	size := len(ip1)
	if size == net.IPv4len {
		a := binary.BigEndian.Uint32(ip1)
		b := binary.BigEndian.Uint32(ip2)
		x := a ^ b
		return uint8(bits.LeadingZeros32(x))
	} else if size == net.IPv6len {
		a := binary.BigEndian.Uint64(ip1)
		b := binary.BigEndian.Uint64(ip2)
		x := a ^ b
		if x != 0 {
			return uint8(bits.LeadingZeros64(x))
		}
		a = binary.BigEndian.Uint64(ip1[8:])
		b = binary.BigEndian.Uint64(ip2[8:])
		x = a ^ b
		return 64 + uint8(bits.LeadingZeros64(x))
	} else {
		err := errors.New("wrong size bit string")
		panic(err)
	}
}

func (node *trieEntry) addToPeerEntries() {
	node.perPeerElem = node.peer.trieEntries.PushBack(node)
}

// removeFromPeerEntries removes a peer's entry table.
func (node *trieEntry) removeFromPeerEntries() {
	if node.perPeerElem != nil {
		node.peer.trieEntries.Remove(node.perPeerElem)
		node.perPeerElem = nil
	}
}

func (node *trieEntry) choose(ip []byte) byte {
	return (ip[node.bitAtByte] >> node.bitAtShift) & 1
}

func (node *trieEntry) maskSelf() {
	mask := net.CIDRMask(int(node.cidr), len(node.bits)*8)
	for i := 0; i < len(mask); i++ {
		node.bits[i] &= mask[i]
	}
}

func (node *trieEntry) zeroizePointers() {
	// Make the garbage collector's life slightly easier
	node.peer = nil
	node.child[0] = nil
	node.child[1] = nil
	node.parent.parentBit = nil
}

func (node *trieEntry) nodePlacement(ip []byte, cidr uint8) (parent *trieEntry, exact bool) {
	for node != nil && node.cidr <= cidr && commonBits(node.bits, ip) >= node.cidr {
		parent = node
		if parent.cidr == cidr {
			exact = true
			return
		}
		bit := node.choose(ip)
		node = node.child[bit]
	}
	return
}

// insert inserts peer information into the entry table for each endpoint.
func (trie parentIndirection) insert(ip []byte, cidr uint8, peer *Peer) {
	if *trie.parentBit == nil {
		node := &trieEntry{
			peer:       peer,
			parent:     trie,
			bits:       ip,
			cidr:       cidr,
			bitAtByte:  cidr / 8,
			bitAtShift: 7 - (cidr % 8),
		}
		node.maskSelf()
		node.addToPeerEntries()
		*trie.parentBit = node

		logger.Debug(fmt.Sprintf("Insert to endpoint entry table=%v", node.bits))

		return
	}

	// If a node entry matching parent exists, delete it and update the existing entry.
	node, exact := (*trie.parentBit).nodePlacement(ip, cidr)
	if exact {
		logger.Debug(fmt.Sprintf("Removing peer entry table=%v", node.bits))
		node.removeFromPeerEntries()
		node.peer = peer
		node.addToPeerEntries()
		return
	}

	newNode := &trieEntry{
		peer:       peer,
		bits:       ip,
		cidr:       cidr,
		bitAtByte:  cidr / 8,
		bitAtShift: 7 - (cidr % 8),
	}
	newNode.maskSelf()
	newNode.addToPeerEntries()

	var down *trieEntry
	if node == nil {
		down = *trie.parentBit
	} else {
		bit := node.choose(ip)
		down = node.child[bit]
		if down == nil {
			newNode.parent = parentIndirection{&node.child[bit], bit}
			node.child[bit] = newNode
			return
		}
	}

	common := commonBits(down.bits, ip)
	if common < cidr {
		cidr = common
	}
	parent := node

	if newNode.cidr == cidr {
		bit := newNode.choose(down.bits)
		down.parent = parentIndirection{&newNode.child[bit], bit}
		newNode.child[bit] = down
		if parent == nil {
			newNode.parent = trie
			*trie.parentBit = newNode
		} else {
			bit := parent.choose(newNode.bits)
			newNode.parent = parentIndirection{&parent.child[bit], bit}
			parent.child[bit] = newNode
		}
		return
	}

	node = &trieEntry{
		bits:       append([]byte{}, newNode.bits...),
		cidr:       cidr,
		bitAtByte:  cidr / 8,
		bitAtShift: 7 - (cidr % 8),
	}
	node.maskSelf()

	bit := node.choose(down.bits)
	down.parent = parentIndirection{&node.child[bit], bit}
	node.child[bit] = down
	bit = node.choose(newNode.bits)
	newNode.parent = parentIndirection{&node.child[bit], bit}
	node.child[bit] = newNode
	if parent == nil {
		node.parent = trie
		*trie.parentBit = node
		logger.Debug(fmt.Sprintf("Insert to new endpoint entry table=%v", newNode.bits))
	} else {
		bit := parent.choose(node.bits)
		node.parent = parentIndirection{&parent.child[bit], bit}
		parent.child[bit] = node
	}
}

// lookup finds the trieEntry table and returns Peer information.
// - ip: specifies the virtual IP address of peer node.
// - node.bits: determine the network address.
func (node *trieEntry) lookup(ip []byte) *Peer {
	var found *Peer
	size := uint8(len(ip))
	for node != nil && commonBits(node.bits, ip) >= node.cidr {
		if node.peer != nil {
			found = node.peer
		}
		if node.bitAtByte == size {
			break
		}
		bit := node.choose(ip)
		node = node.child[bit]
	}
	return found
}

// AllowedIPs manages allowed IP address informations.
// allowedips has an trieEntry tables.
type AllowedIPs struct {
	IPv4  *trieEntry
	IPv6  *trieEntry
	mutex sync.RWMutex
}

// EntriesForPeer entries peer information.
func (table *AllowedIPs) EntriesForPeer(peer *Peer, cb func(prefix netip.Prefix) bool) {
	table.mutex.RLock()
	defer table.mutex.RUnlock()

	for elem := peer.trieEntries.Front(); elem != nil; elem = elem.Next() {
		node := elem.Value.(*trieEntry)
		a, _ := netip.AddrFromSlice(node.bits)
		if !cb(netip.PrefixFrom(a, int(node.cidr))) {
			return
		}
	}
}

// RemoveByPeer remove informations by peer.
func (table *AllowedIPs) RemoveByPeer(peer *Peer) {
	table.mutex.Lock()
	defer table.mutex.Unlock()

	var next *list.Element
	for elem := peer.trieEntries.Front(); elem != nil; elem = next {
		next = elem.Next()
		if node, ok := elem.Value.(*trieEntry); ok {

			node.removeFromPeerEntries()
			node.peer = nil
			if node.child[0] != nil && node.child[1] != nil {
				continue
			}
			bit := 0
			if node.child[0] == nil {
				bit = 1
			}
			child := node.child[bit]
			if child != nil {
				child.parent = node.parent
			}
			*node.parent.parentBit = child
			if node.child[0] != nil || node.child[1] != nil || node.parent.parentBitType > 1 {
				node.zeroizePointers()
				continue
			}
			parent := (*trieEntry)(unsafe.Pointer(uintptr(unsafe.Pointer(node.parent.parentBit)) - unsafe.Offsetof(node.child) - unsafe.Sizeof(node.child[0])*uintptr(node.parent.parentBitType)))
			if parent.peer != nil {
				node.zeroizePointers()
				continue
			}
			child = parent.child[node.parent.parentBitType^1]
			if child != nil {
				child.parent = parent.parent
			}
			*parent.parent.parentBit = child
			node.zeroizePointers()
			parent.zeroizePointers()
		}
	}
}

// Insert interts tables for peer virtual IP addresses.
func (table *AllowedIPs) Insert(prefix netip.Prefix, peer *Peer) error {
	table.mutex.Lock()
	defer table.mutex.Unlock()

	if prefix.Addr().Is6() {
		ip6 := prefix.Addr().As16()
		parentIndirection{&table.IPv6, 2}.insert(ip6[:], uint8(ip.VIPv6HostMatchPrefix), peer)
		return nil
	} else if prefix.Addr().Is4() {
		ip4 := prefix.Addr().As4()
		parentIndirection{&table.IPv4, 2}.insert(ip4[:], uint8(ip.VIPv4HostMatchPrefix), peer)
		return nil
	} else {
		err := errors.New("inserting unknown address type")
		return err
	}
}

// Lookup lookups peer information from allowed IPs table.
func (table *AllowedIPs) Lookup(address []byte) (*Peer, error) {
	table.mutex.RLock()
	defer table.mutex.RUnlock()

	switch len(address) {
	case net.IPv6len:
		return table.IPv6.lookup(address), nil
	case net.IPv4len:
		return table.IPv4.lookup(address), nil
	default:
		err := errors.New("looking up unknown address type")
		return nil, err
	}
}
