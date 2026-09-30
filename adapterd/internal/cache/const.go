package cache

import "time"

const (
	nodeIdCachePeriod         = 0 * time.Second  // infinity
	dnsCachePeriod            = 10 * time.Second // 10s
	routeDirectionCachePeriod = 30 * time.Second // 30s
	pathIdCachePeriod         = 60 * time.Second // 60s
)
