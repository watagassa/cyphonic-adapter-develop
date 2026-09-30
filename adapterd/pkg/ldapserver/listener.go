package ldapserver

import (
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

const acceptRetryDelay = 100 * time.Millisecond

// handshakingListener は Accept 時に TLS ハンドシェイクを実行し、ログを記録するリスナー
//
// ldapserver v1.0.1 は Accept がエラーを返すと nil 接続を参照して panic するため、
// Accept はエラーを返さない。一時的なエラーは再試行し、Close 後はブロックし続ける
type handshakingListener struct {
	net.Listener
	handshakeTimeout time.Duration
	closed           chan struct{}
	closeOnce        sync.Once
}

func newHandshakingListener(ln net.Listener, handshakeTimeout time.Duration) *handshakingListener {
	return &handshakingListener{
		Listener:         ln,
		handshakeTimeout: handshakeTimeout,
		closed:           make(chan struct{}),
	}
}

func (l *handshakingListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			select {
			case <-l.closed:
				// 停止後は serve ループへ戻さない（永久にブロック）
				select {}
			default:
			}

			logger.Warn(fmt.Sprintf("[Listener] Accept failed, retrying: %v", err))
			time.Sleep(acceptRetryDelay)

			continue
		}

		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			return conn, nil
		}

		if err := l.handshake(tlsConn); err != nil {
			// 単一クライアントのエラーで serve ループを落とさないため、次の接続待ちへループ
			continue
		}

		// TLS 復号後のパケットをキャプチャするために loggingConn でラップして返す
		return &loggingConn{Conn: tlsConn}, nil
	}
}

func (l *handshakingListener) Close() error {
	var err error

	l.closeOnce.Do(func() {
		close(l.closed)
		err = l.Listener.Close()
	})

	return err
}

// handshake は TLS ハンドシェイクを明示的に実行する。失敗した場合は接続を閉じてエラーを返す
func (l *handshakingListener) handshake(tlsConn *tls.Conn) error {
	remoteAddr := tlsConn.RemoteAddr().String()
	logger.Info(fmt.Sprintf("[TLS Phase 1] Incoming connection from %s. Starting TLS Handshake...", remoteAddr))

	// ハンドシェイク無応答による無制限ハングアップを防止するためのデッドライン設定
	if l.handshakeTimeout > 0 {
		_ = tlsConn.SetDeadline(time.Now().Add(l.handshakeTimeout))
	}

	if err := tlsConn.Handshake(); err != nil {
		// クライアント側が証明書を出さない等のエラーログを出力し、接続を閉じる
		logger.Warn(fmt.Sprintf("[TLS Error] Handshake failed from %s: %v", remoteAddr, err))

		_ = tlsConn.Close()

		return err
	}

	state := tlsConn.ConnectionState()
	logger.Info(fmt.Sprintf("[TLS Phase 2] Handshake Success with %s | Protocol: %s, CipherSuite: 0x%04x",
		remoteAddr, tlsVersionToString(state.Version), state.CipherSuite))

	if len(state.PeerCertificates) > 0 {
		clientCert := state.PeerCertificates[0]
		logger.Info(fmt.Sprintf("[TLS Auth] Client Certificate Presented by %s | Subject: %s | Serial: %s | Issuer: %s",
			remoteAddr, clientCert.Subject, clientCert.SerialNumber.String(), clientCert.Issuer))
	} else {
		logger.Warn(fmt.Sprintf("[TLS Auth Warning] Handshake completed with %s but NO Client Certificate presented.", remoteAddr))
	}

	// ハンドシェイク成功後、タイムアウト制限を解除
	_ = tlsConn.SetDeadline(time.Time{})

	return nil
}

// loggingConn は Read / Write された生データ（TLS復号後）をログ出力するラッパー。
// ldapserver v1.0.1 にはクライアント単位のデータ保持がないため、Bind 済み DN もここで保持する
type loggingConn struct {
	net.Conn

	mu        sync.Mutex
	boundDN   string
	closeOnce sync.Once
}

func (c *loggingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		logger.Debug(fmt.Sprintf("[Packet Received from FreeRadius (%s) - %d bytes]\n%s",
			c.RemoteAddr(), n, hex.Dump(b[:n])))
	}

	return n, err
}

func (c *loggingConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		logger.Debug(fmt.Sprintf("[Packet Sent to FreeRadius (%s) - %d bytes]\n%s",
			c.RemoteAddr(), n, hex.Dump(b[:n])))
	}

	return n, err
}

func (c *loggingConn) Close() error {
	c.closeOnce.Do(func() {
		boundDN := c.getBoundDN()
		if boundDN == "" {
			boundDN = "Anonymous"
		}

		logger.Info(fmt.Sprintf("[Connection] Session closed for %s (Bound DN: %s)", c.RemoteAddr(), boundDN))
	})

	return c.Conn.Close()
}

func (c *loggingConn) setBoundDN(dn string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.boundDN = dn
}

func (c *loggingConn) getBoundDN() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.boundDN
}

// tlsConnOf は conn（loggingConn でラップされている場合はその中身）から *tls.Conn を取り出す。
// TLS 接続でない場合は nil を返す
func tlsConnOf(conn net.Conn) *tls.Conn {
	if lc, ok := conn.(*loggingConn); ok {
		conn = lc.Conn
	}

	tlsConn, _ := conn.(*tls.Conn)

	return tlsConn
}
