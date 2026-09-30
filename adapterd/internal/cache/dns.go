package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

type dnsInfo struct {
	srcPort          uint16 // srcPort: General nodes DNS listening port.
	dnsTransactionID uint16 // dnstTransactionID: DNS queries Transaction ID.
	expires          int64
}

type DNSCache struct {
	dnsInfos map[uint32]*dnsInfo
	mu       sync.Mutex
}

func NewDNSCache() *DNSCache {
	c := &DNSCache{dnsInfos: make(map[uint32]*dnsInfo)}
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			<-t.C
			c.mu.Lock()
			for k, v := range c.dnsInfos {
				if v.Expired(time.Now().UnixNano()) {
					delete(c.dnsInfos, k)
					logger.Warn(fmt.Sprintf("dnsInfo has expired: DNS TransactionID=%x", v.dnsTransactionID))
				}
			}
			c.mu.Unlock()
		}
	}()

	return c
}

func (c *DNSCache) Put(key uint32, srcPort, dnsTransactionID uint16) {
	c.mu.Lock()

	expires := time.Now().Add(dnsCachePeriod).UnixNano()

	if _, ok := c.dnsInfos[key]; !ok {
		c.dnsInfos[key] = &dnsInfo{
			srcPort:          srcPort,
			dnsTransactionID: dnsTransactionID,
			expires:          expires,
		}
	}

	c.mu.Unlock()
}

func (c *DNSCache) Get(key uint32) (srcPort, dnsTransactionID uint16) {
	c.mu.Lock()

	if v, ok := c.dnsInfos[key]; ok {
		srcPort = v.srcPort
		dnsTransactionID = v.dnsTransactionID
	}

	c.mu.Unlock()

	return srcPort, dnsTransactionID
}

func (d *dnsInfo) Expired(time int64) bool {
	if d.expires == 0 {
		return true
	}

	return time > d.expires
}
