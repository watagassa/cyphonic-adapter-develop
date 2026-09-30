// Package ldapserver provides an LDAPS server with mutual TLS (mTLS) authentication
// that resolves device entries from the client certificate serial number.
package ldapserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	ldap "github.com/vjeantet/ldapserver"
	"go.uber.org/zap"
)

const (
	defaultListenAddr       = "127.0.0.1:636"
	defaultReadTimeout      = 30 * time.Second
	defaultWriteTimeout     = 30 * time.Second
	defaultHandshakeTimeout = 10 * time.Second
)

// Config holds the settings of the LDAPS server.
type Config struct {
	ListenAddr       string // listen address (default: 127.0.0.1:636)
	CertFile         string // server certificate (PEM)
	KeyFile          string // server private key (PEM)
	CAFile           string // CA certificate used to verify client certificates (PEM)
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	HandshakeTimeout time.Duration
}

// Server is an LDAPS server that requires client certificates.
type Server struct {
	cfg    Config
	server *ldap.Server
}

// New creates a Server from cfg. Zero values in cfg are replaced with defaults.
func New(cfg Config) *Server {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}

	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}

	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}

	if cfg.HandshakeTimeout == 0 {
		cfg.HandshakeTimeout = defaultHandshakeTimeout
	}

	return &Server{cfg: cfg}
}

// Start loads the TLS settings, starts listening and serves requests in a new goroutine.
// Serve errors after startup are sent to the returned channel.
func (s *Server) Start() (<-chan error, error) {
	tlsConfig, err := loadTLSConfig(s.cfg.CertFile, s.cfg.KeyFile, s.cfg.CAFile)
	if err != nil {
		return nil, err
	}

	logger.Info("[TLS] Mutual TLS (mTLS) verification STRICTLY enabled.")

	tcpLn, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", s.cfg.ListenAddr, err)
	}

	ldap.Logger = zap.NewStdLog(zap.L())
	s.server = newLDAPServer(s.cfg)

	// 標準の tls.Listener を handshakingListener でラップ
	ln := &handshakingListener{
		Listener:         tls.NewListener(tcpLn, tlsConfig),
		handshakeTimeout: s.cfg.HandshakeTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		logger.Info(fmt.Sprintf("LDAPS Server listening on tls://%s", s.cfg.ListenAddr))

		if err := s.server.Serve(ln); err != nil && !errors.Is(err, net.ErrClosed) {
			errCh <- fmt.Errorf("LDAPS server serve error: %w", err)
		}
	}()

	return errCh, nil
}

// Stop gracefully stops the server.
func (s *Server) Stop() {
	if s.server == nil {
		return
	}

	logger.Info("Shutting down LDAPS Server...")
	s.server.Stop()
}

func newLDAPServer(cfg Config) *ldap.Server {
	server := ldap.NewServer()
	server.ReadTimeout = cfg.ReadTimeout
	server.WriteTimeout = cfg.WriteTimeout

	server.OnNewConnection = func(c net.Conn) error {
		logger.Info(fmt.Sprintf("[Connection] New TCP connection established from: %s", c.RemoteAddr()))

		return nil
	}

	server.OnClientClose = func(conn net.Conn, data any) {
		boundDN, _ := data.(string)
		if boundDN == "" {
			boundDN = "Anonymous"
		}

		logger.Info(fmt.Sprintf("[Connection] Session closed for %s (Bound DN: %s)", conn.RemoteAddr(), boundDN))
	}

	server.Handle(newRouteMux())

	return server
}
