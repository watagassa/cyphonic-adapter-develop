package internal

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/Pluslab/cyphonic-adapter/adapterd/internal/cache"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers/ip"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// RoutineReceiveIncoming receives udp packet from real interfaces.
// - Adapter's packet flow: Peer node -> External interface -> Internal interface -> General node
func (adapter *AdapterDevice) RoutineReceiveIncoming(dnsCache *cache.DNSCache, nodeIDCache *cache.NodeIDCache, routeDirectionCache *cache.RouteDirectionCache, pathIDCache *cache.PathIDCache, dhcpCh chan DHCPQueue) {
	defer func() {
		defer logger.Debug("Routine: Receive Incoming - stopped")
		adapter.queue.decryption.wg.Done()
	}()
	logger.Debug("Routine: Receive Incoming - started")

	buffer := adapter.GetMessageBuffer()
	for {
		adapter.PutMessageBuffer(buffer)
		buffer = adapter.GetMessageBuffer()

		size, peerAddr, err := adapter.ConnUDP.ReadFromUDPAddrPort(buffer[:])
		if err != nil {
			adapter.PutMessageBuffer(buffer)
			buffer = adapter.GetMessageBuffer()
			continue
		}

		if size < 8 {
			continue
		}

		packet := buffer[:size]

		child, err := adapter.LookUpChildDevice(packet, nodeIDCache, pathIDCache)
		if err != nil {
			logger.Error(fmt.Errorf("Routine Receive Incoming error [MessageType=%v]: %w", layers.TypeClass(packet[layers.MessageOffsetPacketType]), err))
			continue
		}

		switch layers.TypeClass(packet[layers.MessageOffsetPacketType]) {
		case layers.TypeClassKeepAlive:
			pathID := hex.EncodeToString(packet[layers.MessageOffsetPathID : layers.MessageOffsetPathID+layers.PathIDlen])
			pathIDCache.Update(pathID)
		case layers.TypeClassRegistrationResponse:
			rres, err := layers.UnmarshalRegistrationResponse(packet, child.commonKey)
			if err != nil {
				logger.Error(fmt.Errorf("failed to unmarshal Registration Response: %w", err))
			}

			logger.Debug("Receive Registration Response", rres)
			child.running.Done() // Reporting completion of child device's Registration Response reception.
		case layers.TypeClassAck:
			ack, err := layers.UnmarshalAck(packet)
			if err != nil {
				logger.Error(fmt.Errorf("failed to unmarshal ACK: %w", err))
			}
			logger.Debug("Receive ACK packet", ack)

			if layers.FlagClass(ack.BaseHeader.Flag) == layers.FlagClassOptimization {
				if peer, ok := child.peers.keyMap[hex.EncodeToString(ack.BaseHeader.ID)]; ok {
					if peer.isTRS.Get() {
						peer.Optimize()
						logger.Debug(fmt.Sprintf("Successfully Route Optimization: ChildDevice=%s", peer.child.deviceName))
					}
				}
			}
		case layers.TypeClassRouteDirectionToCN:
			rd, err := layers.UnmarshalRouteDirection(buffer[:size], child.commonKey)
			if err != nil {
				logger.Error(fmt.Errorf("failed to unmarshal Route Direction to Responder: %w", err))
			}

			if err := validateNodeID(rd.BaseHeader.ID, child.nodeID); err != nil {
				logger.Error(fmt.Errorf("detected inappropriate RouteDirection to Responser BaseHeader.NodeID: %w", err))
			}

			logger.Debug("Receive Route Direction to Responder", rd)
			logger.Debug(fmt.Sprintf("Route Direction to Responder for %s [FQDN=%s][NodeID=%x]", child.deviceName, rd.CNFQDN, rd.BaseHeader.ID))

			logger.Debug(fmt.Sprintf("PathID=%s", hex.EncodeToString(rd.PathID)))
			logger.Debug(fmt.Sprintf("MAC address=%s", child.MacAddress.String()))

			macAddr := pathIDCache.Get(hex.EncodeToString(rd.PathID))
			if macAddr == "" {
				pathIDCache.Put(hex.EncodeToString(rd.PathID), child.MacAddress.String()) // Responder node
			}

			// Info: Store the RouteDirection packet keyed by PathID.
			routeDirectionCache.Put(hex.EncodeToString(rd.PathID), &rd)

			switch rd.ProcessCode {
			case layers.TunnelRequestToTRS: // UDP Hole punching packet is sent to the TRS.
				logger.Debug("Tunel Request to TRS (Hole Punching)")
				var destAddr net.UDPAddr

				hp := layers.HolePunching{}
				hp.GenerateHolePunching(rd.PathID)

				binHolePunching, err := hp.Marshal()
				if err != nil {
					logger.Error(fmt.Errorf("failed to marshal Hole Punching: %w", err))
				}

				switch adapter.ExternalIPVersion {
				case layers.TypeLocalIPVersion4:
					if rd.NATmnIPv4 != ip.ZeroAddr4() {
						destAddr = net.UDPAddr{
							IP:   net.IP(rd.NATmnIPv4.AsSlice()).To4(),
							Port: int(rd.NATmnPort),
						}
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binHolePunching, destAddr.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Hole Punching to %v (Peer): %w", destAddr.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Hole Punching to %v (Peer)", destAddr.AddrPort()))
						}
					} else {
						logger.Debug(fmt.Sprintf("Hole punching will not be sent [NATmnIPv4=%v]", rd.NATmnIPv4))
					}

					destAddr = net.UDPAddr{
						IP:   net.IP(rd.TRSIPv4.AsSlice()).To4(),
						Port: adapter.Cfg.Global.TRSPort,
					}
				case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
					if rd.NATmnIPv6 != ip.ZeroAddr6() {
						destAddr = net.UDPAddr{
							IP:   net.IP(rd.NATmnIPv4.AsSlice()).To16(),
							Port: int(rd.NATmnPort),
						}
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binHolePunching, destAddr.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Hole Punching to %v (Peer): %w", destAddr.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Hole Punching to %v (Peer)", destAddr.AddrPort()))
						}
					} else {
						logger.Debug(fmt.Sprintf("Hole punching will not be sent [NATmnIPv6=%v]", rd.NATmnIPv6))
					}

					destAddr = net.UDPAddr{
						IP:   net.IP(rd.TRSIPv6.AsSlice()).To16(),
						Port: adapter.Cfg.Global.TRSPort,
					}
				}

				if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binHolePunching, destAddr.AddrPort()); err != nil {
					logger.Error(fmt.Errorf("failed to send Hole Punching to %v (Peer): %w", destAddr.AddrPort(), err))
				} else {
					logger.Debug(fmt.Sprintf("Send Hole Punching to %v (TRS)", destAddr.AddrPort()))
				}

			case layers.TunnelRequestToNATcn: // UDP Hole punching packet is sent to the peer NAPT.
				logger.Debug("Tunel Request to NAT Responder (Hole Punching)")
				var destAddr net.UDPAddr

				switch adapter.ExternalIPVersion {
				case layers.TypeLocalIPVersion4:
					destAddr = net.UDPAddr{
						IP:   net.IP(rd.NATmnIPv4.AsSlice()).To4(),
						Port: int(rd.NATmnPort),
					}
				case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
					destAddr = net.UDPAddr{
						IP:   net.IP(rd.NATmnIPv6.AsSlice()).To16(),
						Port: int(rd.NATmnPort),
					}
				}

				hp := layers.HolePunching{}
				hp.GenerateHolePunching(rd.PathID)

				binHolePunching, err := hp.Marshal()
				if err != nil {
					logger.Error(fmt.Errorf("failed to marshal Hole Punching: %w", err))
				}

				if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binHolePunching, destAddr.AddrPort()); err != nil {
					logger.Error(fmt.Errorf("failed to send Hole Punching to %v (Peer): %w", destAddr.AddrPort(), err))
				} else {
					logger.Debug(fmt.Sprintf("Send Hole Punching to %v (Peer)", destAddr.AddrPort()))
				}
			}

			// Check the state of child device.
			if child == nil || child.isClosed() || child.isDown() {
				logger.Warn(fmt.Sprintf("child device is closed: %s", child.deviceName))
				continue
			}

			rd.ChangeRouteDirectionType(layers.TypeClassRouteDirectionConfirmation, child.nodeID)

			logger.Debug("Generate Route Direction Confirmation", rd)

			rdc, err := rd.Marshal(child.commonKey)
			if err != nil {
				logger.Error(fmt.Errorf("failed to marshal Route Direction Confirmation: %w", err))
			}

			switch adapter.ExternalIPVersion {
			case layers.TypeLocalIPVersion4:
				if _, err = adapter.ConnUDP.WriteToUDPAddrPort(rdc, adapter.NMSAddrv4.AddrPort()); err != nil {
					logger.Error(fmt.Errorf("failed to send Route Direction Confirmation to %v: %w", adapter.NMSAddrv4.AddrPort(), err))
				} else {
					logger.Debug(fmt.Sprintf("Send Route Direction Confirmation to %v", adapter.NMSAddrv4.AddrPort()))
				}
			case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
				if _, err = adapter.ConnUDP.WriteToUDPAddrPort(rdc, adapter.NMSAddrv6.AddrPort()); err != nil {
					logger.Error(fmt.Errorf("failed to send Route Direction Confirmation to %v: %w", adapter.NMSAddrv6.AddrPort(), err))
				} else {
					logger.Debug(fmt.Sprintf("Send Route Direction Confirmation to %v", adapter.NMSAddrv6.AddrPort()))
				}
			}

		case layers.TypeClassRouteDirectionToMN:
			rd, err := layers.UnmarshalRouteDirection(packet, child.commonKey)
			if err != nil {
				logger.Error(fmt.Errorf("failed to unmarshal Route Direction to Initiator: %w", err))
			}

			logger.Debug("Receive Route Direction to Initiator", rd)
			logger.Debug(fmt.Sprintf("Route Direction to Initiator for %s [FQDN=%s][PathID=%x]", child.deviceName, rd.CNFQDN, rd.BaseHeader.ID))

			// Info: Store the RouteDirection packet keyed by PathID.
			routeDirectionCache.Put(hex.EncodeToString(rd.PathID), &rd)

			/* Tunnel establish process */
			treq := layers.TunnelRequest{}
			endKey := treq.GenerateTunnelRequest(rd.PathID, rd.TemporaryKey, rd.ProcessCode)

			logger.Debug("Generate Tunnel Request", treq)

			if peer, ok := child.peers.keyMap[hex.EncodeToString(rd.PathID)]; !ok {
				np, err := child.NewPeer(&rd, nodeTypeMN, endKey)
				if err != nil {
					logger.Error(fmt.Errorf("failed to create new %v peer: %w", child.deviceName, err))
				}

				logger.Debug(fmt.Sprintf("Created %s's New Peer: ChildVirtualIPv4=%v  ChildVirtualIPv4=%v PeerVirtualIPv4=%v PeerVirtualIPv6=%v", np.child.deviceName, np.child.VirtualIPv4, np.child.VirtualIPv6, np.dstVirtualIPv4, np.dstVirtualIPv6))

				binTunnelReq, err := treq.Marshal(rd.TunnelKey)
				if err != nil {
					logger.Error(fmt.Errorf("failed to marshal Tunnel Request: %w", err))
				}

				if rd.ProcessCode == layers.TunnelRequestToTRS {
					switch adapter.ExternalIPVersion {
					case layers.TypeLocalIPVersion4:
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelReq, np.endpointTRSv4.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Tunnel Request to %v (TRS): %w", np.endpointTRSv4.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Tunnel Request to %v (TRS)", np.endpointTRSv4.AddrPort()))
						}
					case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelReq, np.endpointTRSv6.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Tunnel Request to %v (TRS): %w", np.endpointTRSv6.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Tunnel Request to %v (TRS)", np.endpointTRSv6.AddrPort()))
						}
					}
				} else {
					switch adapter.ExternalIPVersion {
					case layers.TypeLocalIPVersion4:
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelReq, np.endpointv4.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Tunnel Request to %v (Peer): %w", np.endpointv4.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Tunnel Request to %v (Peer)", np.endpointv4.AddrPort()))
						}
					case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelReq, np.endpointv6.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Tunnel Request to %v (Peer): %w", np.endpointv6.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Tunnel Request to %v (Peer)", np.endpointv6.AddrPort()))
						}
					}
				}
			} else {
				peer.Update(&rd, nodeTypeMN, endKey)

				logger.Debug(fmt.Sprintf("Updated %s's Peer: ChildVirtualIPv4=%v  ChildVirtualIPv4=%v PeerVirtualIPv4=%v PeerVirtualIPv6=%v", peer.child.deviceName, peer.child.VirtualIPv4, peer.child.VirtualIPv6, peer.dstVirtualIPv4, peer.dstVirtualIPv6))

				binTunnelReq, err := treq.Marshal(rd.TunnelKey)
				if err != nil {
					logger.Error(fmt.Errorf("failed to marshal Tunnel Request: %w", err))
				}

				if rd.ProcessCode == layers.TunnelRequestToTRS {
					switch adapter.ExternalIPVersion {
					case layers.TypeLocalIPVersion4:
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelReq, peer.endpointTRSv4.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Tunnel Request to %v (TRS): %w", peer.endpointTRSv4.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Tunnel Request to %v (TRS)", peer.endpointTRSv4.AddrPort()))
						}
					case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelReq, peer.endpointTRSv6.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Tunnel Request to %v (TRS): %w", peer.endpointTRSv6.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Tunnel Request to %v (TRS)", peer.endpointTRSv6.AddrPort()))
						}
					}
				} else {
					switch adapter.ExternalIPVersion {
					case layers.TypeLocalIPVersion4:
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelReq, peer.endpointv4.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Tunnel Request to %v (Peer): %w", peer.endpointv4.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Tunnel Request to %v (Peer)", peer.endpointv4.AddrPort()))
						}
					case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
						if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelReq, peer.endpointv6.AddrPort()); err != nil {
							logger.Error(fmt.Errorf("failed to send Tunnel Request to %v (Peer): %w", peer.endpointv6.AddrPort(), err))
						} else {
							logger.Debug(fmt.Sprintf("Send Tunnel Request to %v (Peer)", peer.endpointv6.AddrPort()))
						}
					}
				}
			}

		case layers.TypeClassHolePunching:
			hp, err := layers.UnmarshalHolePunching(packet)
			if err != nil {
				logger.Error(fmt.Errorf("failed to unmarshal Hole Punching: %w", err))
			}

			logger.Debug("Receive Hole Punching", hp)

			if layers.FlagClass(hp.BaseHeader.Flag) == layers.FlagClassOptimization {
				if peer, ok := child.peers.keyMap[hex.EncodeToString(hp.BaseHeader.ID)]; ok {
					if peer.isTRS.Get() {
						peer.Optimize()
						ack := layers.Ack{}
						ack.GenerateAck(hp.BaseHeader.ID)
						layers.SerializeFlag(&ack.BaseHeader, layers.FlagClassOptimization)
						binAck, err := ack.Marshal()
						if err != nil {
							logger.Error(fmt.Errorf("failed to marshal ACK: %w", err))
						}

						if _, err := adapter.ConnUDP.WriteToUDPAddrPort(binAck, peerAddr); err != nil {
							logger.Error(fmt.Errorf("failed to send ACK to %v: %w", peerAddr, err))
						} else {
							logger.Debug(fmt.Sprintf("Send ACK to %v", peerAddr))
							logger.Debug(fmt.Sprintf("Successfully Route Optimization: ChildDevice=%s", peer.child.deviceName))
						}
					}
				}
			}

		case layers.TypeClassTunnelRequest:
			rd := routeDirectionCache.Get(hex.EncodeToString(buffer[layers.MessageOffsetPathID : layers.MessageOffsetPathID+layers.PathIDlen]))

			treq, err := layers.UnmarshalTunnelRequest(packet, rd.TunnelKey)
			if err != nil {
				logger.Error(fmt.Errorf("failed to unmarshal Tunnel Request: %w", err))
				continue
			}

			logger.Debug(fmt.Sprintf("Receive Tunnel Request from %v", peerAddr))

			if peer, ok := child.peers.keyMap[hex.EncodeToString(rd.PathID)]; !ok {
				var np *Peer

				if rd.ProcessCode == layers.TunnelRequestToTRS {
					dk, err := treq.DecryptEndKey(rd.TemporaryKey)
					if err != nil {
						logger.Error(fmt.Errorf("failed to decrypt end key: %w", err))
					}

					np, err = child.NewPeer(&rd, nodeTypeCN, dk)
					if err != nil {
						logger.Error(fmt.Errorf("failed to %s' create new Peer: %w", child.deviceName, err))
					} else {
						logger.Debug(fmt.Sprintf("Created %s's New Peer: ChildVirtualIPv4=%v  ChildVirtualIPv4=%v PeerVirtualIPv4=%v PeerVirtualIPv6=%v", np.child.deviceName, np.child.VirtualIPv4, np.child.VirtualIPv6, np.dstVirtualIPv4, np.dstVirtualIPv6))
					}
				} else {
					np, err = child.NewPeer(&rd, nodeTypeCN, treq.EndKey)
					if err != nil {
						logger.Error(fmt.Errorf("failed to %s' create new Peer: %w", child.deviceName, err))
					} else {
						logger.Debug(fmt.Sprintf("Created %s's New Peer: ChildVirtualIPv4=%v  ChildVirtualIPv4=%v PeerVirtualIPv4=%v PeerVirtualIPv6=%v", np.child.deviceName, np.child.VirtualIPv4, np.child.VirtualIPv6, np.dstVirtualIPv4, np.dstVirtualIPv6))
					}
				}

				tres := layers.TunnelResponse{}
				tres.GenerateTunnelResponse(treq.BaseHeader.ID)

				logger.Debug("Generate Tunnel Response", tres)

				binTunnelRes, err := tres.Marshal()
				if err != nil {
					logger.Error(fmt.Errorf("failed to marshal Tunnel Response: %w", err))
				}

				// TODO: It is necessary to provide a flag to determine whether it is addressed to TRS or MN.
				if peerAddr.Addr().Unmap() == np.endpointTRSv4.AddrPort().Addr() {
					if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelRes, np.endpointTRSv4.AddrPort()); err != nil {
						logger.Error(fmt.Errorf("failed to send Tunnel Response to %v (TRS): %w", np.endpointTRSv4.AddrPort(), err))
					} else {
						logger.Debug(fmt.Sprintf("Send Tunnel Response to %v (TRS)", np.endpointTRSv4.AddrPort()))
					}
				} else if peerAddr.Addr() == np.endpointTRSv6.AddrPort().Addr() {
					if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelRes, np.endpointTRSv4.AddrPort()); err != nil {
						logger.Error(fmt.Errorf("failed to send Tunnel Response to %v (TRS): %w", np.endpointTRSv4.AddrPort(), err))
					} else {
						logger.Debug(fmt.Sprintf("Send Tunnel Response to %v (TRS)", np.endpointTRSv6.AddrPort()))
					}
				} else {
					if _, err := adapter.ConnUDP.WriteToUDPAddrPort(binTunnelRes, peerAddr); err != nil {
						logger.Error(fmt.Errorf("failed to send Tunnel Response to %v (Peer): %w", peerAddr, err))
					} else {
						logger.Debug(fmt.Sprintf("Send Tunnel Response to %v (Peer)", peerAddr))
					}
				}
			} else {
				if rd.ProcessCode == layers.TunnelRequestToTRS {
					dk, err := treq.DecryptEndKey(rd.TemporaryKey)
					if err != nil {
						logger.Error(fmt.Errorf("failed to decrypt end key: %w", err))
					}

					peer.Update(&rd, nodeTypeCN, dk)
				} else {
					peer.Update(&rd, nodeTypeCN, treq.EndKey)
				}

				tres := layers.TunnelResponse{}
				tres.GenerateTunnelResponse(treq.BaseHeader.ID)

				logger.Debug("Generate Tunnel Response", tres)

				binTunnelRes, err := tres.Marshal()
				if err != nil {
					logger.Error(fmt.Errorf("failed to marshal Tunnel Response: %w", err))
				}

				// TODO: It is necessary to provide a flag to determine whether it is addressed to TRS or MN.
				if peerAddr.Addr().Unmap() == peer.endpointTRSv4.AddrPort().Addr() {
					if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelRes, peer.endpointTRSv4.AddrPort()); err != nil {
						logger.Error(fmt.Errorf("failed to send Tunnel Response to %v (TRS): %w", peer.endpointTRSv4.AddrPort(), err))
					} else {
						logger.Debug(fmt.Sprintf("Send Tunnel Response to %v (TRS)", peer.endpointTRSv4.AddrPort()))
					}
				} else if peerAddr.Addr() == peer.endpointTRSv6.AddrPort().Addr() {
					if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binTunnelRes, peer.endpointTRSv6.AddrPort()); err != nil {
						logger.Error(fmt.Errorf("failed to send Tunnel Response to %v (TRS): %w", peer.endpointTRSv6.AddrPort(), err))
					} else {
						logger.Debug(fmt.Sprintf("Send Tunnel Response to %v (TRS)", peer.endpointTRSv6.AddrPort()))
					}
				} else {
					if _, err := adapter.ConnUDP.WriteToUDPAddrPort(binTunnelRes, peerAddr); err != nil {
						logger.Error(fmt.Errorf("failed to send Tunnel Response to %v (Peer): %w", peerAddr, err))
					} else {
						logger.Debug(fmt.Sprintf("Send Tunnel Response to %v (Peer)", peerAddr))
					}
				}
			}

		case layers.TypeClassTunnelResponse:
			tres, err := layers.UnmarshalTunnelResponse(packet)
			if err != nil {
				continue
			}

			logger.Debug(fmt.Sprintf("Receive Tunnel Response from %v", peerAddr))

			// If Tunnel Response is received from TRS: Attempt route optimization using UDP Hole Punching.
			if peer, ok := child.peers.keyMap[hex.EncodeToString(tres.BaseHeader.ID)]; ok {
				logger.Debug(fmt.Sprintf("Receive Tunnel Response for %s (%s)", peer.child.deviceName, peer.child.fqdn))

				if peer.isTRS.Get() {
					hp := layers.HolePunching{}
					hp.GenerateHolePunching(tres.BaseHeader.ID)
					layers.SerializeFlag(&hp.BaseHeader, layers.FlagClassOptimization)
					binHolePunching, err := hp.Marshal()
					if err != nil {
						logger.Error(fmt.Errorf("failed to marshal Hole Punching: %w", err))
					}

					switch adapter.ExternalIPVersion {
					case layers.TypeLocalIPVersion4:
						if peer.endpointv4.AddrPort().Addr() != ip.ZeroAddr4() {
							if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binHolePunching, peer.endpointv4.AddrPort()); err != nil {
								logger.Error(fmt.Errorf("failed to send Hole Punching to %v (Peer): %w", peer.endpointv4.AddrPort(), err))
							} else {
								logger.Debug(fmt.Sprintf("Send Hole Punching to %v (Peer)", peer.endpointv4.AddrPort()))
							}
						} else {
							logger.Debug(fmt.Sprintf("Hole punching will not be sent [Peer=%v]", peer.endpointv4.AddrPort().Addr()))
						}
					case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
						if peer.endpointv6.AddrPort().Addr() != ip.ZeroAddr6() {
							if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binHolePunching, peer.endpointv6.AddrPort()); err != nil {
								logger.Error(fmt.Errorf("failed to send Hole Punching to %v (Peer): %w", peer.endpointv6.AddrPort(), err))
							} else {
								logger.Debug(fmt.Sprintf("Send Hole Punching to %v (Peer)", peer.endpointv6.AddrPort()))
							}
						} else {
							logger.Debug(fmt.Sprintf("Hole punching will not be sent [Peer=%v]", peer.endpointv6.AddrPort().Addr()))
						}
					}
				}
			}

			// Get RouteDirection cache information.
			rd := routeDirectionCache.Get(hex.EncodeToString(tres.BaseHeader.ID))

			/* DNS response */
			switch child.isVirtualIPv6.Get() {
			case false:
				dstPort, dnstid := dnsCache.Get(rd.BaseHeader.TransactionID)
				dnsPacket, err := adapter.AnswerARecordForChildDevice(dstPort, dnstid, string(rd.CNFQDN), net.IP(rd.MNVirtualIPv4.AsSlice()).To4(), net.IP(rd.CNVirtualIPv4.AsSlice()).To4())
				if err != nil {
					logger.Error(fmt.Errorf("failed to generate DNS answer A record: %w", err))
				}

				if err := SendPacket4(adapter.socket.sendIPv4, dnsPacket, net.IP(child.VirtualIPv4.AsSlice()).To4()); err != nil {
					logger.Error(fmt.Errorf("failed to send DNS answer A record to %v: %w", child.VirtualIPv4, err))
				} else {
					logger.Debug(fmt.Sprintf("Send DNS answer A record [ChildDevice=%s] [TargetFQDN=%s] [VirtualIPv4=%v]", child.deviceName, string(rd.CNFQDN), rd.CNVirtualIPv4))
				}
			case true:
				dstPort, dnstid := dnsCache.Get(rd.BaseHeader.TransactionID)
				dnsPacket, err := adapter.AnswerAAAARecordForChildDevice(dstPort, dnstid, string(rd.CNFQDN), net.IP(rd.MNVirtualIPv6.AsSlice()).To16(), net.IP(rd.CNVirtualIPv6.AsSlice()).To16())
				if err != nil {
					logger.Error(fmt.Errorf("failed to generate DNS answer A record: %w", err))
				}

				if err := SendPacket6(adapter.socket.sendIPv6, dnsPacket, net.IP(child.VirtualIPv6.AsSlice()).To16()); err != nil {
					logger.Error(fmt.Errorf("failed to send DNS answer AAAA record to %v: %w", child.VirtualIPv6, err))
				} else {
					logger.Debug(fmt.Sprintf("Send DNS answer AAAA record [ChildDevice=%s] [TargetFQDN=%s] [VirtualIPv6=%v]", child.deviceName, string(rd.CNFQDN), rd.CNVirtualIPv6))
				}
			}
		case layers.TypeClassCapsuleMessage:
			pathID := hex.EncodeToString(packet[layers.MessageOffsetPathID : layers.MessageOffsetPathID+layers.PathIDlen])

			if peer, ok := child.peers.keyMap[pathID]; ok {
				pathIDCache.Update(pathID)

				elem := adapter.GetInboundElement()
				elem.key = peer.endKey
				elem.packet = packet
				elem.buffer = buffer
				elem.Mutex = sync.Mutex{}
				elem.Lock()

				// add to decryption queues
				if peer.isRunning.Get() {
					peer.queue.inbound.c <- elem
					peer.child.Adapter.queue.decryption.c <- elem
					buffer = adapter.GetMessageBuffer()
				} else {
					adapter.PutInboundElement(elem)
				}
			}
		}
	}
}

// LookUpChildDevice finds a general node.
// This function is executed when a generic node sends any signaling.
// Get the MAC address of the general node, using this address as a key
// to get the general node inforamtion from the internal cache
// https://cyphonic.esa.io/posts/51
// - Registration process: get the general node's mac address using NodeID as a key.
// - Direction process and after: Obtains the mac address of the general node using the PathID as a key.
func (adapter *AdapterDevice) LookUpChildDevice(packet []byte, nodeIDCache *cache.NodeIDCache, pathIDCache *cache.PathIDCache) (*ChildDevice, error) {
	var child *ChildDevice
	var ok bool

	if layers.TypeClass(packet[layers.MessageOffsetPacketType]) == layers.TypeClassRegistrationResponse || layers.TypeClass(packet[layers.MessageOffsetPacketType]) == layers.TypeClassRouteDirectionToCN {
		/* NodeIDCache => RegistrationResponse(7) / RouteDirectionToResponder(12) */

		// Find general node cache information.
		macAddr := nodeIDCache.Get(hex.EncodeToString(packet[layers.MessageOffsetNodeID : layers.MessageOffsetNodeID+layers.NodeIDlen]))
		if macAddr == "" {
			err := errors.New("failed to get child device MAC address")
			return child, err
		}

		if child, ok = adapter.children.keyMap[macAddr]; !ok {
			err := errors.New("failed to get child device")
			return child, err
		}
	} else {
		/* PathIDCache => RouteDirectionToInitiator(14) / TunnelRequest(22) / TunnelResponse(23) / CapsuleMessage(24) / KeepAlive(8) */

		// Find general node cache information.
		macAddr := pathIDCache.Get(hex.EncodeToString(packet[layers.MessageOffsetPathID : layers.MessageOffsetPathID+layers.PathIDlen]))
		if macAddr == "" {
			err := errors.New("failed to get child device MAC address")
			return child, err
		}

		if child, ok = adapter.children.keyMap[macAddr]; !ok {
			err := errors.New("failed to get child device")
			return child, err
		}
	}

	return child, nil
}

// validateNodeID checks the NodeID included in RouteDirectionToResponder's NodeID
// and its own NodeID present in the device structure.
func validateNodeID(routeDirectionToResponderNodeID, deviceNodeID []byte) error {
	if !bytes.Equal(routeDirectionToResponderNodeID, deviceNodeID) {
		return fmt.Errorf("detected invalid NodeID: %x", routeDirectionToResponderNodeID)
	}

	return nil
}
