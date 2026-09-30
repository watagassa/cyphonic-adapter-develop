package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

type pathIDInfo struct {
	macAddr string // macAddr: ChildDevice's MAC address.
	expires int64
}

type PathIDCache struct {
	pathIDInfos map[string]*pathIDInfo
	mu          sync.Mutex
}

func NewPathIDCache() *PathIDCache {
	c := &PathIDCache{pathIDInfos: make(map[string]*pathIDInfo)}
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			<-t.C
			c.mu.Lock()
			for k, v := range c.pathIDInfos {
				if v.Expired(time.Now().UnixNano()) {
					delete(c.pathIDInfos, k)
					logger.Warn(fmt.Sprintf("pathIDInfo has expired: MAC address=%s", v.macAddr))
				}
			}
			c.mu.Unlock()
		}
	}()

	return c
}

func (c *PathIDCache) Put(key string, macAddr string) {
	c.mu.Lock()

	expires := time.Now().Add(pathIdCachePeriod).UnixNano()

	if _, ok := c.pathIDInfos[key]; !ok {
		c.pathIDInfos[key] = &pathIDInfo{
			macAddr: macAddr,
			expires: expires,
		}
	}

	c.mu.Unlock()
}

func (c *PathIDCache) Get(key string) (macAddr string) {
	c.mu.Lock()
	if v, ok := c.pathIDInfos[key]; ok {
		macAddr = v.macAddr
	}
	c.mu.Unlock()

	return macAddr
}

func (c *PathIDCache) Update(key string) {
	c.mu.Lock()
	if v, ok := c.pathIDInfos[key]; ok {
		v.expires = time.Now().Add(pathIdCachePeriod).UnixNano()
	}
	c.mu.Unlock()
}

func (d *pathIDInfo) Expired(time int64) bool {
	if d.expires == 0 {
		return true
	}
	return time > d.expires
}
