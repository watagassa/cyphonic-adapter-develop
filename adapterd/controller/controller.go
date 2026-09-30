package controller

import (
	"fmt"

	"github.com/Pluslab/cyphonic-adapter/adapterd/config"
	"github.com/Pluslab/cyphonic-adapter/adapterd/domain"
	"github.com/Pluslab/cyphonic-adapter/adapterd/internal"
	"github.com/Pluslab/cyphonic-adapter/adapterd/internal/cache"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// Run reads the config from the configuration file (config.yaml or config.yml)
// and runs the main routine.
func Run() (domain.StatusCode, error) {
	cfg, err := config.Get()
	if err != nil {
		return domain.ProcessStatusCodeConfigError, fmt.Errorf("failed to load config file: %w", err)
	}

	domain.StartingMessage()

	logger.InitLogger(cfg)
	logger.Debug(cfg.Global)
	logger.Debug(cfg.Adapterd)
	logger.Debug(cfg.Logging)

	if err := execute(cfg); err != nil {
		return domain.ProcessStatusCodeAbnormalExit, fmt.Errorf("adapter daemon has been stopped: %w", err)
	}

	logger.Info("Gracefully terminates the adapter device...")

	return domain.ProcessStatusCodeNoError, nil
}

// execute creates a channel, internal cache, and initializes the AdapterDevice.
// Then, receiving the trigger signal from the channel and passes it to the processing job.
func execute(cfg *config.Config) error {
	var adapter internal.AdapterDevice
	adapter.Cfg = cfg

	switch cfg.Adapterd.VirtualIPType {
	case "4":
		domain.Version4Message()
		// create channel
		dnsCh := make(chan internal.DNSQueue, 64)
		defer close(dnsCh)
		dhcpCh := make(chan internal.DHCPQueue, 64)
		defer close(dhcpCh)
		updateInfoCh := make(chan bool, 64)
		defer close(updateInfoCh)

		// create internal cache
		nodeIDCache := cache.NewNodeIDCache()
		dnsCache := cache.NewDNSCache()
		routeDirectionCache := cache.NewRouteDirectionCache()
		pathIDCache := cache.NewPathIDCache()

		if err := adapter.NewAdapterDevice(dnsCh, dnsCache, dhcpCh); err != nil {
			return fmt.Errorf("failed to create new adapter device: %w", err)
		}

		go adapter.RoutineReceiveIncoming(dnsCache, nodeIDCache, routeDirectionCache, pathIDCache, dhcpCh)

		go adapter.RecoverAllChildDevices(nodeIDCache, cfg.Adapterd.VirtualIPType)

		go func() {
			for {
				select {
				// General nodes connected.
				case dhcpMsg := <-dhcpCh:
					adapter.NewChildDevice(dhcpMsg, nodeIDCache)
				// General nodes initiate communication to peers.
				case dnsMsg := <-dnsCh:
					adapter.HandleChildDeviceDirection(dnsMsg, pathIDCache)
				// Update general node information.
				case <-updateInfoCh:
					adapter.UpdateChildDeviceInformation()
				}
			}
		}()

		if err := adapter.CloseAdapterDevice(); err != nil {
			return fmt.Errorf("adapter device is not safely terminated: %w", err)
		}

		return nil
	case "6":
		domain.Version6Message()
		// create channel
		dnsCh := make(chan internal.DNSQueue, 64)
		defer close(dnsCh)
		dhcpCh := make(chan internal.DHCPQueue, 64)
		defer close(dhcpCh)

		// create internal cache
		nodeIDCache := cache.NewNodeIDCache()
		dnsCache := cache.NewDNSCache()
		routeDirectionCache := cache.NewRouteDirectionCache()
		pathIDCache := cache.NewPathIDCache()

		if err := adapter.NewAdapterDevice(dnsCh, dnsCache, dhcpCh); err != nil {
			return fmt.Errorf("failed to create new adapter device: %w", err)
		}

		go adapter.RoutineReceiveIncoming(dnsCache, nodeIDCache, routeDirectionCache, pathIDCache, dhcpCh)

		go adapter.RecoverAllChildDevices(nodeIDCache, cfg.Adapterd.VirtualIPType)

		go func() {
			for {
				select {
				// General nodes connected.
				case dhcpMsg := <-dhcpCh:
					adapter.NewChildDevice(dhcpMsg, nodeIDCache)
				// General nodes initiate communication to peers.
				case dnsMsg := <-dnsCh:
					adapter.HandleChildDeviceDirection(dnsMsg, pathIDCache)
				}
			}
		}()

		if err := adapter.CloseAdapterDevice(); err != nil {
			return fmt.Errorf("adapter device is not safely terminated: %w", err)
		}

		return nil
	}

	return fmt.Errorf("unsupported virtual IP type: %s", cfg.Adapterd.VirtualIPType)
}
