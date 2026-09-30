package internal

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Pluslab/cyphonic-adapter/adapterd/internal/cache"
	"github.com/Pluslab/cyphonic-adapter/adapterd/layers"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// HandleAuthentication handles the CYPHONIC authentication process.
func HandleAuthentication(conn net.Conn, authElem *layers.AuthenticationElement) layers.LoginResponse {
	var err error
	lres := layers.LoginResponse{}
	ok := false

	defer func() {
		if err := conn.Close(); err != nil {
			logger.Error(fmt.Errorf("failed to close tls connection: %w", err))
		}
	}()

	// No progress until a login response is received.
	for !ok {
		lres, ok, err = handleLoginRequest(conn, authElem)
		if err != nil {
			logger.Error(fmt.Errorf("failed to execute authentication process: %w", err))
		}
	}

	return lres
}

// handleLoginRequest sends a LoginRequest packet to the AS,
// and returns a LoginResponse packet on successful authentication.
func handleLoginRequest(conn net.Conn, authElem *layers.AuthenticationElement) (layers.LoginResponse, bool, error) {
	var valiableLength uint16
	var err error
	var lreq layers.LoginRequest
	var lres layers.LoginResponse
	var isReturnResponse bool

	isReturnResponse = false
	buf := make([]byte, MTU)

	lreq.HandleLoginPattern(authElem)

	if err := layers.SerializeBaseHeader(&lreq.BaseHeader); err != nil {
		return lres, isReturnResponse, fmt.Errorf("failed to serialize BaseHeader: %w", err)
	}

	layers.SerializeType(&lreq.BaseHeader, layers.TypeClassLoginRequest)
	switch lreq.AuthType {
	case layers.TypeDeviceIDPassword:
		valiableLength = lreq.DeviceIDLength + lreq.PasswordLength
	case layers.TypeDigitalAuthentication:
		valiableLength = lreq.CertificateLength
	case layers.TypeSingleSignOn:
	}

	layers.SerializeLoginRequestPacketLength(&lreq, valiableLength)

	logger.Debug("Generate Login Request", lreq)

	b, err := lreq.Marshal()
	if err != nil {
		return lres, isReturnResponse, fmt.Errorf("failed to marshal Login Request: %w", err)
	}

	bi := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	if _, err = bi.Write(b); err != nil {
		return lres, isReturnResponse, fmt.Errorf("failed to send Login Request: %w", err)
	}

	logger.Debug(fmt.Sprintf("Send Login Request to %v", conn.RemoteAddr()))

	if err = bi.Flush(); err != nil {
		return lres, isReturnResponse, fmt.Errorf("failed to flush bytes: %w", err)
	}

	if _, err = bi.Read(buf); err != nil {
		if errors.Is(err, io.EOF) {
			return lres, isReturnResponse, fmt.Errorf("failed to read received packet: %w", err)
		}
		return lres, isReturnResponse, fmt.Errorf("failed to read Login Response: %w", err)
	}

	if !layers.IsLoginResponse(buf) {
		return lres, isReturnResponse, fmt.Errorf("Login Response is invalid: %w", err)
	}

	lres, err = layers.UnmarshalLoginResponse(buf)
	if err != nil {
		return lres, isReturnResponse, fmt.Errorf("failed to unmarshal Login Response: %w", err)
	}
	isReturnResponse = true

	logger.Debug("Receive Login Response", lres)

	return lres, isReturnResponse, nil
}

// HandleRegistration handles the CYPHONIC registration process.
func (adapter *AdapterDevice) HandleRegistration() error {
CreateRegist:
	var err error
	var rreq layers.RegistrationRequest
	var rres layers.RegistrationResponse

	adapter.ExternalIPVersion, err = layers.GenerateRegistrationRequest(&rreq, adapter.nodeID, adapter.ConnUDP.LocalAddr(), adapter.Cfg.Adapterd.DnsTunAddress, adapter.Cfg.Adapterd.DnsTun6Address)
	if err != nil {
		return fmt.Errorf("failed to generate Registration Request: %w", err)
	}

	logger.Debug("Generate Registration Request", rreq)

	binRegistReq, err := rreq.Marshal(adapter.commonKey)
	if err != nil {
		return fmt.Errorf("failed to marshal Registration Request: %w", err)
	}

	switch adapter.ExternalIPVersion {
	case layers.TypeLocalIPNone:
		goto CreateRegist
	case layers.TypeLocalIPVersion4:
		if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binRegistReq, adapter.NMSAddrv4.AddrPort()); err != nil {
			return fmt.Errorf("failed to send Registration Request (%v): %w", adapter.NMSAddrv4.AddrPort(), err)
		}
		logger.Debug(fmt.Sprintf("Send Registration Request to %v", adapter.NMSAddrv4.AddrPort()))
	case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
		if _, err = adapter.ConnUDP.WriteToUDPAddrPort(binRegistReq, adapter.NMSAddrv6.AddrPort()); err != nil {
			return fmt.Errorf("failed to send Registration Request (%v): %w", adapter.NMSAddrv6.AddrPort(), err)
		}
		logger.Debug(fmt.Sprintf("Send Registration Request to %v", adapter.NMSAddrv6.AddrPort()))
	}

	buf := make([]byte, MTU)
	if _, err := adapter.ConnUDP.Read(buf); err != nil {
		if errors.Is(err, io.EOF) {
			return err
		}
		return fmt.Errorf("failed to read Registration Response: %w", err)
	}

	rres, err = layers.UnmarshalRegistrationResponse(buf, adapter.commonKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Registration Response: %w", err)
	}

	logger.Debug("Receive Registration Response", rres)

	logger.Debug(fmt.Sprintf("NAPT Flag=%t", rres.NATFlag != 0))

	return nil
}

// HandleChildDeviceRegistration handles the CYPHONIC registration process for general node.
func (child *ChildDevice) HandleChildDeviceRegistration(nodeIDCache *cache.NodeIDCache) error {
CreateRegist:
	var err error
	var rreq layers.RegistrationRequest

	child.Adapter.ExternalIPVersion, err = layers.GenerateRegistrationRequest(&rreq, child.nodeID, child.Adapter.ConnUDP.LocalAddr(), child.Adapter.Cfg.Adapterd.DnsTunAddress, child.Adapter.Cfg.Adapterd.DnsTun6Address)
	if err != nil {
		return fmt.Errorf("failed to generate Registration Request: %w", err)
	}

	logger.Debug("Generate Registration Request", rreq)

	binRegistReq, err := rreq.Marshal(child.commonKey)
	if err != nil {
		return fmt.Errorf("failed to marshal Registration Request: %w", err)
	}

	switch child.Adapter.ExternalIPVersion {
	case layers.TypeLocalIPNone:
		goto CreateRegist
	case layers.TypeLocalIPVersion4:
		if _, err = child.Adapter.ConnUDP.WriteToUDPAddrPort(binRegistReq, child.Adapter.NMSAddrv4.AddrPort()); err != nil {
			return fmt.Errorf("failed to send Registration Request (%v): %w", child.Adapter.NMSAddrv4.AddrPort(), err)
		}
		logger.Debug(fmt.Sprintf("Send Registration Request to %v", child.Adapter.NMSAddrv4.AddrPort()))
	case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
		if _, err = child.Adapter.ConnUDP.WriteToUDPAddrPort(binRegistReq, child.Adapter.NMSAddrv6.AddrPort()); err != nil {
			return fmt.Errorf("failed to send Registration Request (%v): %w", child.Adapter.NMSAddrv6.AddrPort(), err)
		}
		logger.Debug(fmt.Sprintf("Send Registration Request to %v", child.Adapter.NMSAddrv6.AddrPort()))
	}

	// Puts Registration Cache
	nodeIDCache.Put(hex.EncodeToString(child.nodeID), child.MacAddress.String())

	return nil
}

// HandleAcquisition handles the CYPHONIC adapter General node Information Acquisition process.
func (adapter *AdapterDevice) HandleAcquisition(httpClient *http.Client) error {
	baseUrl := fmt.Sprintf("https://%s:%d", adapter.Cfg.Global.ControllerFQDN, adapter.Cfg.Global.ControllerPort)

	if err := healthz(httpClient, baseUrl); err != nil {
		return fmt.Errorf("failed to health check to %s: %w", baseUrl, err)
	}

	endPoint := baseUrl + "/api/adapter/" + adapter.Cfg.Adapterd.AdapterID + "/children"
	ares, err := layers.AcquisitionRequest(httpClient, endPoint)
	if err != nil {
		return fmt.Errorf("failed to send Acquisition Request to %v: %w", ares, err)
	}

	if err := layers.ChildDeviceInformationSave(ares, adapter.Cfg.Adapterd.ChildDeviceInformationPath); err != nil {
		return fmt.Errorf("failed to save child device information: %w", err)
	}

	if err = adapter.ReadChildDeviceInformation(); err != nil {
		return fmt.Errorf("failed to read child device information: %w", err)
	}

	return nil
}

// HandleChildDeviceDirection handles the CYPHONIC direction process for general node.
// When it receives a communication request from a general node, it executes route selection processing.
func (adapter *AdapterDevice) HandleChildDeviceDirection(dnsMsg DNSQueue, pathIDCache *cache.PathIDCache) {
	dr := layers.DirectionRequest{}

	logger.Debug(fmt.Sprintf("Get DNS Request packet - deviceName=%s", dnsMsg.child.deviceName))
	logger.Debug(fmt.Sprintf("Get DNS Request packet - srcFQDN=%s", string(dnsMsg.child.fqdn)))
	logger.Debug(fmt.Sprintf("Get DNS Request packet - dstFQDN=%s", string(dnsMsg.dstFQDN)))

	macAddress := dnsMsg.child.MacAddress
	nodeID := dnsMsg.child.nodeID
	commonKey := dnsMsg.child.commonKey

	layers.GenerateDirectionRequest(&dr, nodeID, dnsMsg.child.fqdn, dnsMsg.dstFQDN)

	dr.BaseHeader.TransactionID = dnsMsg.transactionID

	bdr, err := dr.Marshal(commonKey)
	if err != nil {
		logger.Error(fmt.Errorf("failed to marshal Direction Request: %w", err))
	}

	switch adapter.ExternalIPVersion {
	case layers.TypeLocalIPVersion4:
		if _, err = adapter.ConnUDP.WriteToUDPAddrPort(bdr, adapter.NMSAddrv4.AddrPort()); err != nil {
			logger.Error(fmt.Errorf("failed to send Direction Request to %v: %w", adapter.NMSAddrv4.AddrPort(), err))
		} else {
			logger.Debug(fmt.Sprintf("Send Direction Request to %v", adapter.NMSAddrv4.AddrPort()))
		}
	case layers.TypeLocalIPVersion6, layers.TypeLocalDualStackNetwork:
		if _, err = adapter.ConnUDP.WriteToUDPAddrPort(bdr, adapter.NMSAddrv6.AddrPort()); err != nil {
			logger.Error(fmt.Errorf("failed to send Direction Request to %v: %w", adapter.NMSAddrv6.AddrPort(), err))
		} else {
			logger.Debug(fmt.Sprintf("Send Direction Request to %v", adapter.NMSAddrv6.AddrPort()))
		}
	}

	logger.Debug(fmt.Sprintf("PathID=%s", hex.EncodeToString(dr.PathID)))
	logger.Debug(fmt.Sprintf("MAC address=%s", macAddress.String()))

	macAddr := pathIDCache.Get(hex.EncodeToString(dr.PathID))
	if macAddr == "" {
		pathIDCache.Put(hex.EncodeToString(dr.PathID), macAddress.String()) // Initiator node
	}
}

func healthz(httpClient *http.Client, baseUrl string) error {
	res, err := httpClient.Get(baseUrl + "/healthz")
	if err != nil {
		return fmt.Errorf("failed to get health check endpoint: %w", err)
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			logger.Error(fmt.Errorf("failed to close health check response body: %w", closeErr))
		}
	}()

	if res.StatusCode != 200 {
		return fmt.Errorf("terminated with a status code other than 200: status=%d", res.StatusCode)
	}

	return nil
}

// TODO: Implement HandleAcquisitionById.
// HandleAcquisitionById executes the General node Information Acquisition process
// by specifying the device ID of the child device.
func (adapter *AdapterDevice) HandleAcquisitionById() {
}

// TODO: Implement UpdateChildDeviceInformation.
// UpdateChildDeviceInformation updates ChildDevice information if general node information has changed.
func (adapter *AdapterDevice) UpdateChildDeviceInformation() {
}
