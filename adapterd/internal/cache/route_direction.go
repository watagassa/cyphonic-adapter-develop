package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

type routeDirectionInfo struct {
	routeDirection []byte // routeDirection: RouteDirectionToInitiator packet.
	expires        int64
}

type RouteDirectionCache struct {
	routeDirectionInfos map[string]*routeDirectionInfo
	mu                  sync.Mutex
}

func NewRouteDirectionCache() *RouteDirectionCache {
	c := &RouteDirectionCache{routeDirectionInfos: make(map[string]*routeDirectionInfo)}
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			<-t.C
			c.mu.Lock()
			for k, v := range c.routeDirectionInfos {
				if v.Expired(time.Now().UnixNano()) {
					delete(c.routeDirectionInfos, k)
					logger.Warn(fmt.Sprintf("routeDirectionInfo has expired: Route Direction Packet"))
				}
			}
			c.mu.Unlock()
		}
	}()

	return c
}

func (c *RouteDirectionCache) Put(key string, routeDirection *layers.RouteDirection) {
	c.mu.Lock()

	expires := time.Now().Add(routeDirectionCachePeriod).UnixNano()

	bytes, err := json.Marshal(&routeDirection)
	if err != nil {
		logger.Error(fmt.Errorf("failed to json marshal Route Direction: %w", err))
	}

	rd := make([]byte, len(bytes))
	copy(rd, bytes)
	c.routeDirectionInfos[key] = &routeDirectionInfo{
		routeDirection: rd,
		expires:        expires,
	}

	c.mu.Unlock()
}

func (c *RouteDirectionCache) Get(key string) (routeDirection layers.RouteDirection) {
	rd := layers.RouteDirection{}
	c.mu.Lock()
	if v, ok := c.routeDirectionInfos[key]; ok {
		err := json.Unmarshal(v.routeDirection, &rd)
		if err != nil {
			logger.Error(fmt.Errorf("failed to json unmarshal Route Direction: %w", err))
			return rd
		}
		routeDirection = rd
	}
	c.mu.Unlock()

	return routeDirection
}

func (d *routeDirectionInfo) Expired(time int64) bool {
	if d.expires == 0 {
		return true
	}

	return time > d.expires
}
