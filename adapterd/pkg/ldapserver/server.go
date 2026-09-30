// Package ldapserver provides an LDAPS server with mutual TLS (mTLS) authentication
// that resolves device entries from the client certificate serial number.
package ldapserver

import (
	"crypto/tls"
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
	cfg      Config
	server   *ldap.Server
	listener *handshakingListener
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
// It returns after the listener is ready, or with an error if listening failed.
func (s *Server) Start() error {
	tlsConfig, err := loadTLSConfig(s.cfg.CertFile, s.cfg.KeyFile, s.cfg.CAFile)
	if err != nil {
		return err
	}

	logger.Info("[TLS] Mutual TLS (mTLS) verification STRICTLY enabled.")

	ldap.Logger = zap.NewStdLog(zap.L())
	s.server = newLDAPServer(s.cfg)

	ready := make(chan struct{})
	errCh := make(chan error, 1)

	// ldapserver v1.0.1 には既存リスナーを渡す Serve がないため、
	// ListenAndServe のオプションで TCP リスナーを TLS + handshakingListener に差し替える
	wrapListener := func(srv *ldap.Server) {
		s.listener = newHandshakingListener(tls.NewListener(srv.Listener, tlsConfig), s.cfg.HandshakeTimeout)
		srv.Listener = s.listener

		logger.Info(fmt.Sprintf("LDAPS Server listening on tls://%s", s.cfg.ListenAddr))
		close(ready)
	}

	go func() {
		if err := s.server.ListenAndServe(s.cfg.ListenAddr, wrapListener); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ready:
		return nil
	case err := <-errCh:
		s.server = nil

		return fmt.Errorf("failed to listen on %s: %w", s.cfg.ListenAddr, err)
	}
}

// Stop gracefully closes client connections and the listener.
//
// ldapserver v1.0.1 panics when Accept returns an error, so after Stop the serve
// goroutine stays blocked in Accept instead of exiting.
func (s *Server) Stop() {
	if s.server == nil {
		return
	}

	logger.Info("Shutting down LDAPS Server...")
	s.server.Stop()

	if err := s.listener.Close(); err != nil {
		logger.Warn(fmt.Sprintf("failed to close LDAPS listener: %v", err))
	}

	s.server = nil
}

func newLDAPServer(cfg Config) *ldap.Server {
	server := ldap.NewServer()
	server.ReadTimeout = cfg.ReadTimeout
	server.WriteTimeout = cfg.WriteTimeout

	server.Handle(newRouteMux())

	server.OnNewConnection = func(c net.Conn) error {
		logger.Info(fmt.Sprintf("[Connection] New TCP connection established from: %s", c.RemoteAddr()))

		return nil
	}

	return server
}
