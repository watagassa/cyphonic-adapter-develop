package internal

import (
	"sync"
	"sync/atomic"
)

// MaxMessageSize is 65535=(2^16)-1.
const MaxMessageSize = (1 << Bit16) - 1

// WaitPool pools wait pooling.
type WaitPool struct {
	pool  sync.Pool
	cond  sync.Cond
	lock  sync.Mutex
	count uint32
	max   uint32
}

// NewWaitPool initializes wait pool.
func NewWaitPool(max uint32, newFunc func() interface{}) *WaitPool {
	p := &WaitPool{pool: sync.Pool{New: newFunc}, max: max}
	p.cond = sync.Cond{L: &p.lock}

	return p
}

// Get selects an arbitrary item from the Pool, removes it from the
// Pool, and returns it to the caller.
func (p *WaitPool) Get() interface{} {
	if p.max != 0 {
		p.lock.Lock()
		for atomic.LoadUint32(&p.count) >= p.max {
			p.cond.Wait()
		}
		atomic.AddUint32(&p.count, 1)
		p.lock.Unlock()
	}

	return p.pool.Get()
}

// Put adds x to the pool.
func (p *WaitPool) Put(x interface{}) {
	p.pool.Put(x)

	if p.max == 0 {
		return
	}

	atomic.AddUint32(&p.count, ^uint32(0))
	p.cond.Signal()
}

// PopulatePools creates new buffers.
// - message buffers, inbound elements, outbound element, arp element, dns element, dhcp element
func (adapter *AdapterDevice) PopulatePools() {
	adapter.pool.messageBuffers = NewWaitPool(PreallocatedBuffersPerPool, func() interface{} {
		return new([MaxMessageSize]byte)
	})
	adapter.pool.inboundElements = NewWaitPool(PreallocatedBuffersPerPool, func() interface{} {
		return new(QueueInboundElement)
	})
	adapter.pool.outboundElements = NewWaitPool(PreallocatedBuffersPerPool, func() interface{} {
		return new(QueueOutboundElement)
	})
	adapter.pool.arpElements = NewWaitPool(PreallocatedBuffersPerPool, func() interface{} {
		return new(QueueARPElement)
	})
	adapter.pool.dnsElements = NewWaitPool(PreallocatedBuffersPerPool, func() interface{} {
		return new(QueueDNSElement)
	})
	adapter.pool.dhcpElements = NewWaitPool(PreallocatedBuffersPerPool, func() interface{} {
		return new(QueueDHCPElement)
	})
	adapter.pool.ndpElements = NewWaitPool(PreallocatedBuffersPerPool, func() interface{} {
		return new(QueueNDPElement)
	})
}

// GetMessageBuffer gets 65535bytes.
func (adapter *AdapterDevice) GetMessageBuffer() *[MaxMessageSize]byte {
	return adapter.pool.messageBuffers.Get().(*[MaxMessageSize]byte)
}

// PutMessageBuffer puts 65535bytes.
func (adapter *AdapterDevice) PutMessageBuffer(msg *[MaxMessageSize]byte) {
	adapter.pool.messageBuffers.Put(msg)
}

// GetInboundElement gets QueueInboundElement.
func (adapter *AdapterDevice) GetInboundElement() *QueueInboundElement {
	return adapter.pool.inboundElements.Get().(*QueueInboundElement)
}

// PutInboundElement puts QueueInboundElement.
func (adapter *AdapterDevice) PutInboundElement(elem *QueueInboundElement) {
	elem.clearPointers()
	adapter.pool.inboundElements.Put(elem)
}

// GetOutboundElement gets QueueOutboundElement.
func (adapter *AdapterDevice) GetOutboundElement() *QueueOutboundElement {
	return adapter.pool.outboundElements.Get().(*QueueOutboundElement)
}

// PutOutboundElement puts QueueOutboundElement.
func (adapter *AdapterDevice) PutOutboundElement(elem *QueueOutboundElement) {
	elem.clearPointers()
	adapter.pool.outboundElements.Put(elem)
}

func (adapter *AdapterDevice) GetARPElement() *QueueARPElement {
	return adapter.pool.arpElements.Get().(*QueueARPElement)
}

func (adapter *AdapterDevice) PutARPElement(elem *QueueARPElement) {
	elem.clearPointers()
	adapter.pool.arpElements.Put(elem)
}

func (adapter *AdapterDevice) GetDNSElement() *QueueDNSElement {
	return adapter.pool.dnsElements.Get().(*QueueDNSElement)
}

func (adapter *AdapterDevice) PutDNSElement(elem *QueueDNSElement) {
	elem.clearPointers()
	adapter.pool.dnsElements.Put(elem)
}

func (adapter *AdapterDevice) GetNDPElement() *QueueNDPElement {
	return adapter.pool.ndpElements.Get().(*QueueNDPElement)
}

func (adapter *AdapterDevice) PutNDPElement(elem *QueueNDPElement) {
	elem.clearPointers()
	adapter.pool.ndpElements.Put(elem)
}
