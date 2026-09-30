package internal

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
)

// MakeUDPConn returns UDP connection.
// The adapter listens for requests from the peer nodes on UDP 30000 port.
func MakeUDPConn() (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", ":30000")
	if err != nil {
		return nil, fmt.Errorf("failed to get udp listen address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen udp socket: %w", err)
	}

	return conn, nil
}

// MakeTLSConn returns TLS connection.
// The adapter first establishes TLS connection with Authentication Service.
// After that, the adapter also establishes TLS connection when the general nodes authentication.
func (adapter *AdapterDevice) MakeTLSConn() (*tls.Conn, error) {
	CAPool := x509.NewCertPool()

	caCert, err := os.ReadFile(adapter.Cfg.Adapterd.RootCertificatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read root CA certificate: %w", err)
	}

	if ok := CAPool.AppendCertsFromPEM(caCert); !ok {
		err := errors.New("failed to get root CA certificate")
		return nil, err
	}

	clientCert, err := tls.LoadX509KeyPair(adapter.Cfg.Adapterd.TlsClientCertificatePath, adapter.Cfg.Adapterd.TlsClientCertificatePrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get client certificate: %w", err)
	}

	config := &tls.Config{
		RootCAs:      CAPool,
		Certificates: []tls.Certificate{clientCert},
	}

	conn, err := tls.Dial("tcp", adapter.Cfg.Global.ASFQDN+":"+strconv.Itoa(adapter.Cfg.Global.ASPort), config)
	if err != nil {
		return nil, fmt.Errorf("failed to tls dialing: %w", err)
	}

	return conn, nil
}

// MakeHTTPClient returns HTTP client for Controller communication
func (adapter *AdapterDevice) MakeHTTPClient() (*http.Client, error) {
	caCert, err := os.ReadFile(adapter.Cfg.Adapterd.RootCertificatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read root CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to append root CA certificate")
	}

	clientCert, err := tls.LoadX509KeyPair(
		adapter.Cfg.Adapterd.TlsClientCertificatePath,
		adapter.Cfg.Adapterd.TlsClientCertificatePrivateKeyPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caPool,
				Certificates: []tls.Certificate{clientCert},
				ServerName:   adapter.Cfg.Global.ControllerFQDN,
				MinVersion:   tls.VersionTLS13,
			},
		},
	}, nil
}
