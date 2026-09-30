package internal

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/Pluslab/cyphonic-adapter/adapterd/internal/cache"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/google/gopacket"
	golayers "github.com/google/gopacket/layers"
	"github.com/miekg/dns"
)

const domainName = "cyphonic.org"

// DNSElement
type QueueDNSElement struct {
	sync.Mutex
	buffer *[MaxMessageSize]byte
	packet []byte
	udp    *golayers.UDP
	child  *ChildDevice
}

func (adapter *AdapterDevice) NewDNSElement() *QueueDNSElement {
	elem := adapter.GetDNSElement()
	elem.buffer = adapter.GetMessageBuffer()
	elem.Mutex = sync.Mutex{}
	elem.udp = &golayers.UDP{}

	return elem
}

func (elem *QueueDNSElement) clearPointers() {
	elem.buffer = nil
	elem.packet = nil
	elem.udp = nil
}

// DNSQueue is the queue of dns.
type DNSQueue struct {
	transactionID uint32
	dstFQDN       []byte
	child         *ChildDevice
}

// DNSProcessStage stages all DNS packets.
func (adapter *AdapterDevice) DNSProcessStage(elem *QueueDNSElement) {
	for {
		select {
		case adapter.adapterQueue.dns.c <- elem:
			return
		default:
		}
	}
}

// RoutineAnalysisDNS analyzes DNS packets received from general nodes.
// This function works routinely.
func (adapter *AdapterDevice) RoutineAnalysisDNS(dnsCh chan DNSQueue, dnsCache *cache.DNSCache) {
	defer func() {
		adapter.state.stopping.Done()
		logger.Debug("Routine: AnalysisDNS - stopped")
	}()
	logger.Debug("Routine: AnalysisDNS - started")

	for elem := range adapter.adapterQueue.dns.c {
		elem.Lock() // Lock for DNS analysis

		msg := new(dns.Msg)

		if err := msg.Unpack(elem.udp.Payload); err != nil {
			logger.Error(fmt.Errorf("failed to unpack DNS message: %w", err))
		}

		for _, v := range msg.Question {
			switch adapter.child.isVirtualIPv6.Get() {
			case false:
				if v.Qtype == dns.TypeA && strings.Contains(v.Name, domainName) {
					dstFQDN := bytes.TrimRight([]byte(v.Name), ".")
					tid, err := layers.GenerateTransactionID()
					if err != nil {
						logger.Error(fmt.Errorf("failed to generate DNS transactionID: %w", err))
					}

					peer, state := elem.child.tunnelCache.Get(string(dstFQDN))
					switch state {
					case TunnelStateEstablished:
						// If the peer already exists, it will not send a Direction Request and will only respond with a DNS A record.
						dnsPacket, err := adapter.AnswerARecordForChildDevice(uint16(elem.udp.SrcPort), msg.Id, string(dstFQDN), net.IP(elem.child.VirtualIPv4.AsSlice()).To4(), net.IP(peer.dstVirtualIPv4.AsSlice()).To4())
						if err != nil {
							logger.Error(fmt.Errorf("failed to generate DNS answer A record: %w", err))
						}

						if err := SendPacket4(adapter.socket.sendIPv4, dnsPacket, net.IP(elem.child.VirtualIPv4.AsSlice()).To4()); err != nil {
							logger.Error(fmt.Errorf("failed to send DNS answer A record to %v: %w", elem.child.VirtualIPv4, err))
						} else {
							logger.Debug(fmt.Sprintf("Send DNS answer A record [ChildDevice=%s] [TargetFQDN=%s] [VirtualIPv4=%v]", elem.child.deviceName, string(dstFQDN), peer.dstVirtualIPv4))
						}
					case TunnelStatePending:
						// Direction Request already sent, waiting for Direction Response. Do nothing.
						logger.Debug(fmt.Sprintf("Direction Request already pending for [ChildDevice=%s] [TargetFQDN=%s]", elem.child.deviceName, string(dstFQDN)))
					default:
						// No entry exists, register as pending and send Direction Request.
						elem.child.tunnelCache.PutPending(string(dstFQDN))
						dnsCache.Put(tid, uint16(elem.udp.SrcPort), msg.Id)
						dnsCh <- DNSQueue{
							transactionID: tid,
							dstFQDN:       dstFQDN,
							child:         elem.child,
						}
					}
				}
			case true:
				if v.Qtype == dns.TypeAAAA && strings.Contains(v.Name, domainName) {
					dstFQDN := bytes.TrimRight([]byte(v.Name), ".")
					tid, err := layers.GenerateTransactionID()
					if err != nil {
						logger.Error(fmt.Errorf("failed to generate DNS transactionID: %w", err))
					}

					peer, state := elem.child.tunnelCache.Get(string(dstFQDN))
					switch state {
					case TunnelStateEstablished:
						// If the peer already exists, it will not send a Direction Request and will only respond with a DNS AAAA record.
						dnsPacket, err := adapter.AnswerAAAARecordForChildDevice(uint16(elem.udp.SrcPort), msg.Id, string(dstFQDN), net.IP(elem.child.VirtualIPv6.AsSlice()).To16(), net.IP(peer.dstVirtualIPv6.AsSlice()).To16())
						if err != nil {
							logger.Error(fmt.Errorf("failed to generate DNS answer AAAA record: %w", err))
						}

						if err := SendPacket6(adapter.socket.sendIPv6, dnsPacket, net.IP(elem.child.VirtualIPv6.AsSlice()).To16()); err != nil {
							logger.Error(fmt.Errorf("failed to send DNS answer AAAA record to %v: %w", elem.child.VirtualIPv6, err))
						} else {
							logger.Debug(fmt.Sprintf("Send DNS answer AAAA record [ChildDevice=%s] [TargetFQDN=%s] [VirtualIPv6=%v]", elem.child.deviceName, string(dstFQDN), peer.dstVirtualIPv6))
						}
					case TunnelStatePending:
						// Direction Request already sent, waiting for Direction Response. Do nothing.
						logger.Debug(fmt.Sprintf("Direction Request already pending for [ChildDevice=%s] [TargetFQDN=%s]", elem.child.deviceName, string(dstFQDN)))
					default:
						// No entry exists, register as pending and send Direction Request.
						elem.child.tunnelCache.PutPending(string(dstFQDN))
						dnsCache.Put(tid, uint16(elem.udp.SrcPort), msg.Id)
						dnsCh <- DNSQueue{
							transactionID: tid,
							dstFQDN:       dstFQDN,
							child:         elem.child,
						}
					}
				}
			}
		}
		elem.Unlock() // UnLock for DNS analysis
	}
}

// AnswerARecordForChildDevice returns A record to general node.
func (adapter *AdapterDevice) AnswerARecordForChildDevice(dstPort, dnstid uint16, fqdn string, mnVirtualIPv4, cnVirtualIPv4 net.IP) ([]byte, error) {
	msg := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Response:           true,
			Opcode:             0,
			Authoritative:      false,
			Truncated:          false,
			RecursionDesired:   true,
			RecursionAvailable: true,
			Zero:               false,
			AuthenticatedData:  false,
			CheckingDisabled:   false,
			Rcode:              0,
		},
		Compress: false,
		Question: []dns.Question{},
		Answer:   []dns.RR{},
		Ns:       []dns.RR{},
		Extra:    []dns.RR{},
	}

	msg.Id = dnstid

	question := dns.Question{
		Name:   fqdn + ".",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}

	msg.Question = append(msg.Question, question)

	resp := new(dns.A)
	resp.Hdr = dns.RR_Header{
		Name:   fqdn + ".",
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
		Ttl:    600,
	}

	resp.A = cnVirtualIPv4

	msg.Answer = append(msg.Answer, resp)

	payload, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack DNS message: %w", err)
	}

	ip := golayers.IPv4{
		Version:  4,
		Protocol: golayers.IPProtocolUDP,
		SrcIP:    net.IP(adapter.InternalInterface.Addr4.AsSlice()).To4(),
		DstIP:    mnVirtualIPv4,
	}

	udp := golayers.UDP{
		SrcPort: 53,
		DstPort: golayers.UDPPort(dstPort),
	}

	if err := udp.SetNetworkLayerForChecksum(&ip); err != nil {
		return nil, fmt.Errorf("failed to set network layer for checksum: %w", err)
	}

	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	buffer := gopacket.NewSerializeBuffer()

	if err := gopacket.SerializeLayers(buffer, options,
		&ip,
		&udp,
		gopacket.Payload(payload),
	); err != nil {
		return nil, fmt.Errorf("failed to selialize DNS response packet: %w", err)
	}

	outgoingPacket := buffer.Bytes()

	return outgoingPacket, nil
}

// AnswerAAAARecordForChildDevice returns AAAA record to general node.
func (adapter *AdapterDevice) AnswerAAAARecordForChildDevice(dstPort, dnstid uint16, fqdn string, mnVirtualIPv6, cnVirtualIPv6 net.IP) ([]byte, error) {
	msg := dns.Msg{
		MsgHdr: dns.MsgHdr{
			Response:           true,
			Opcode:             0,
			Authoritative:      false,
			Truncated:          false,
			RecursionDesired:   true,
			RecursionAvailable: true,
			Zero:               false,
			AuthenticatedData:  false,
			CheckingDisabled:   false,
			Rcode:              0,
		},
		Compress: false,
		Question: []dns.Question{},
		Answer:   []dns.RR{},
		Ns:       []dns.RR{},
		Extra:    []dns.RR{},
	}

	msg.Id = dnstid

	question := dns.Question{
		Name:   fqdn + ".",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	}

	msg.Question = append(msg.Question, question)

	resp := new(dns.AAAA)
	resp.Hdr = dns.RR_Header{
		Name:   fqdn + ".",
		Rrtype: dns.TypeAAAA,
		Class:  dns.ClassINET,
		Ttl:    600,
	}

	resp.AAAA = cnVirtualIPv6

	msg.Answer = append(msg.Answer, resp)

	payload, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack DNS message: %w", err)
	}

	ip := golayers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       0,
		NextHeader:   golayers.IPProtocolUDP,
		HopLimit:     64,
		SrcIP:        net.IP(adapter.InternalInterface.Addr6.AsSlice()).To16(),
		DstIP:        mnVirtualIPv6,
	}

	udp := golayers.UDP{
		SrcPort: 53,
		DstPort: golayers.UDPPort(dstPort),
	}

	if err := udp.SetNetworkLayerForChecksum(&ip); err != nil {
		return nil, fmt.Errorf("failed to set network layer for checksum: %w", err)
	}

	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	buffer := gopacket.NewSerializeBuffer()

	if err := gopacket.SerializeLayers(buffer, options,
		&ip,
		&udp,
		gopacket.Payload(payload),
	); err != nil {
		return nil, fmt.Errorf("failed to selialize DNS response packet: %w", err)
	}

	outgoingPacket := buffer.Bytes()

	return outgoingPacket, nil
}
