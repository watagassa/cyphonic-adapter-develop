package internal

import (
	"runtime"
	"sync"
)

// Bit16 is bit length, QueueSize defines the size of queue.
const (
	Bit16                      = 16
	QueueOutboundSize          = 1024
	QueueInboundSize           = 1024
	QueueARPSize               = 1024
	QueueDNSSize               = 1024
	QueueDHCPSize              = 1024
	QueueNDPSize               = 1024
	PreallocatedBuffersPerPool = 0 // Disable and allow for infinite memory growth
)

type nodeType uint32

const (
	nodeTypeMN nodeType = iota
	nodeTypeCN
	nodeTypeLocalMN
	nodeTypeLocalCN
)

// An outboundQueue is a channel of QueueOutboundElements awaiting encryption.
// An outboundQueue is ref-counted using its wg field.
// An outboundQueue created with newOutboundQueue has one reference.
// Every additional writer must call wg.Add(1).
// Every completed writer must call wg.Done().
// When no further writers will be added,
// call wg.Done to remove the initial reference.
// When the refcount hits 0, the queue's channel is closed.
type outboundQueue struct {
	c  chan *QueueOutboundElement
	wg sync.WaitGroup
}

func newOutboundQueue() *outboundQueue {
	q := &outboundQueue{
		c: make(chan *QueueOutboundElement, QueueOutboundSize),
	}
	q.wg.Add(1)

	go func() {
		q.wg.Wait()
		close(q.c)
	}()

	return q
}

// A inboundQueue is similar to an outboundQueue; see those docs.
type inboundQueue struct {
	c  chan *QueueInboundElement
	wg sync.WaitGroup
}

func newInboundQueue() *inboundQueue {
	q := &inboundQueue{
		c: make(chan *QueueInboundElement, QueueInboundSize),
	}
	q.wg.Add(1)

	go func() {
		q.wg.Wait()
		close(q.c)
	}()

	return q
}

type autodrainingInboundQueue struct {
	c chan *QueueInboundElement
}

func newAutodrainingInboundQueue(adapter *AdapterDevice) *autodrainingInboundQueue {
	q := &autodrainingInboundQueue{
		c: make(chan *QueueInboundElement, QueueInboundSize),
	}
	runtime.SetFinalizer(q, adapter.flushInboundQueue)

	return q
}

func (adapter *AdapterDevice) flushInboundQueue(q *autodrainingInboundQueue) {
	for {
		select {
		case elem := <-q.c:
			elem.Lock()
			adapter.PutMessageBuffer(elem.buffer)
			adapter.PutInboundElement(elem)
		default:
			return
		}
	}
}

type autodrainingOutboundQueue struct {
	c chan *QueueOutboundElement
}

func newAutodrainingOutboundQueue(adapter *AdapterDevice) *autodrainingOutboundQueue {
	q := &autodrainingOutboundQueue{
		c: make(chan *QueueOutboundElement, QueueOutboundSize),
	}
	runtime.SetFinalizer(q, adapter.flushOutboundQueue)

	return q
}

func (adapter *AdapterDevice) flushOutboundQueue(q *autodrainingOutboundQueue) {
	for {
		select {
		case elem := <-q.c:
			elem.Lock()
			adapter.PutMessageBuffer(elem.buffer)
			adapter.PutOutboundElement(elem)
		default:
			return
		}
	}
}

// autodraining ARP queue
type autodrainingARPQueue struct {
	c chan *QueueARPElement
}

func newAutodrainingARPQueue(adapter *AdapterDevice) *autodrainingARPQueue {
	q := &autodrainingARPQueue{
		c: make(chan *QueueARPElement, QueueARPSize),
	}
	runtime.SetFinalizer(q, adapter.flushARPQueue)

	return q
}

func (adapter *AdapterDevice) flushARPQueue(q *autodrainingARPQueue) {
	for {
		select {
		case elem := <-q.c:
			elem.Lock()
			adapter.PutMessageBuffer(elem.buffer)
			adapter.PutARPElement(elem)
		default:
			return
		}
	}
}

// autodraining DNS queue
type autodrainingDNSQueue struct {
	c chan *QueueDNSElement
}

func newAutodrainingDNSQueue(adapter *AdapterDevice) *autodrainingDNSQueue {
	q := &autodrainingDNSQueue{
		c: make(chan *QueueDNSElement, QueueDNSSize),
	}
	runtime.SetFinalizer(q, adapter.flushDNSQueue)

	return q
}

func (adapter *AdapterDevice) flushDNSQueue(q *autodrainingDNSQueue) {
	for {
		select {
		case elem := <-q.c:
			elem.Lock()
			adapter.PutMessageBuffer(elem.buffer)
			adapter.PutDNSElement(elem)
		default:
			return
		}
	}
}

// autodraining DHCP queue
type autodrainingDHCPQueue struct {
	c chan *QueueDHCPElement
}

func newAutodrainingDHCPQueue(adapter *AdapterDevice) *autodrainingDHCPQueue {
	q := &autodrainingDHCPQueue{
		c: make(chan *QueueDHCPElement, QueueDHCPSize),
	}
	runtime.SetFinalizer(q, adapter.flushDHCPQueue)

	return q
}

func (adapter *AdapterDevice) flushDHCPQueue(q *autodrainingDHCPQueue) {
	for {
		select {
		case dhcp := <-q.c:
			dhcp.Lock()
			adapter.PutDHCPElement(dhcp)
		default:
			return
		}
	}
}

// autodraining NDP queue
type autodrainingNDPQueue struct {
	c chan *QueueNDPElement
}

func newAutodrainingNDPQueue(adapter *AdapterDevice) *autodrainingNDPQueue {
	q := &autodrainingNDPQueue{
		c: make(chan *QueueNDPElement, QueueNDPSize),
	}
	runtime.SetFinalizer(q, adapter.flushNDPQueue)

	return q
}

func (adapter *AdapterDevice) flushNDPQueue(q *autodrainingNDPQueue) {
	for {
		select {
		case elem := <-q.c:
			elem.Lock()
			adapter.PutMessageBuffer(elem.buffer)
			adapter.PutNDPElement(elem)
		default:
			return
		}
	}
}
