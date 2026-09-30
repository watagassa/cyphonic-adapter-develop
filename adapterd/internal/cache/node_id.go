package cache

import "sync"

type nodeIDInfo struct {
	macAddr string // macAddr: ChildDevice's MAC address.
}

type NodeIDCache struct {
	nodeIDInfos map[string]*nodeIDInfo
	mu          sync.Mutex
}

func NewNodeIDCache() *NodeIDCache {
	c := &NodeIDCache{nodeIDInfos: make(map[string]*nodeIDInfo)}
	return c
}

func (c *NodeIDCache) Put(key string, macAddr string) {
	c.mu.Lock()

	if _, ok := c.nodeIDInfos[key]; !ok {
		c.nodeIDInfos[key] = &nodeIDInfo{
			macAddr: macAddr,
		}
	}

	c.mu.Unlock()
}

func (c *NodeIDCache) Get(key string) (macAddr string) {
	c.mu.Lock()

	if v, ok := c.nodeIDInfos[key]; ok {
		macAddr = v.macAddr
	}

	c.mu.Unlock()

	return macAddr
}
