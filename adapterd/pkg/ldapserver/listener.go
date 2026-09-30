package ldapserver

import (
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// handshakingListener は Accept 時に TLS ハンドシェイクを実行し、ログを記録するリスナー
type handshakingListener struct {
	net.Listener
	handshakeTimeout time.Duration
}

func (l *handshakingListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			return conn, nil
		}

		if err := l.handshake(tlsConn); err != nil {
			// 単一クライアントのエラーで Serve() を落とさないため、次の接続待ちへループ
			continue
		}

		// TLS 復号後のパケットをキャプチャするために loggingConn でラップして返す
		return &loggingConn{Conn: tlsConn}, nil
	}
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

// loggingConn は Read / Write された生データ（TLS復号後）をログ出力するラッパー
type loggingConn struct {
	net.Conn
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

// tlsConnOf は conn（loggingConn でラップされている場合はその中身）から *tls.Conn を取り出す。
// TLS 接続でない場合は nil を返す
func tlsConnOf(conn net.Conn) *tls.Conn {
	if lc, ok := conn.(*loggingConn); ok {
		conn = lc.Conn
	}

	tlsConn, _ := conn.(*tls.Conn)

	return tlsConn
}
