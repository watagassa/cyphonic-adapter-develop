package cmd

import (
	"fmt"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/go-ping/ping"
)

const (
	HealthCheckPingRequestInterval = 1 * time.Second // every 1 second
	HealthCheckPingRequestTimeout  = 3 * time.Second // timeout after 3 seconds
)

// executePingCmd sends a ping to a general node.
func ExecutePingCmd(targetAddr, sourceAddr string, count int) (*ping.Statistics, error) {
	pinger, err := ping.NewPinger(targetAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Pinger: %w", err)
	}
	pinger.Interval = HealthCheckPingRequestInterval
	pinger.Timeout = HealthCheckPingRequestTimeout
	pinger.Count = count
	pinger.Source = sourceAddr

	if err = pinger.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute ping command: %w", err)
	}

	logger.Debug(fmt.Sprintf("ping command executed to %s", targetAddr))

	stats := pinger.Statistics()

	return stats, nil
}
