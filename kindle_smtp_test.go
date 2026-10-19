package zlib

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// smtpSession records what the fake SMTP server saw.
type smtpSession struct {
	authOverTLS bool
	from, to    string
	data        string
}

// fakeSMTP is a minimal SMTP server on 127.0.0.1. implicitTLS makes it
// handshake before the greeting (port 465 style); offerSTARTTLS advertises
// STARTTLS on a plain connection (port 587 style).
type fakeSMTP struct {
	port     int
	sessions chan smtpSession
}

// smtpTestTLS returns a server config for 127.0.0.1 and points the client's
// root pool at its certificate.
func smtpTestTLS(t *testing.T) *tls.Config {
	t.Helper()
	srv := httptest.NewUnstartedServer(nil)
	srv.StartTLS()
	cfg := srv.TLS.Clone()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	srv.Close()

	original := smtpRootCAs
	smtpRootCAs = pool
	t.Cleanup(func() { smtpRootCAs = original })
	return cfg
}

func startFakeSMTP(t *testing.T, tlsConfig *tls.Config, implicitTLS, offerSTARTTLS bool) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	f := &fakeSMTP{port: ln.Addr().(*net.TCPAddr).Port, sessions: make(chan smtpSession, 1)}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		if implicitTLS {
			conn = tls.Server(conn, tlsConfig)
		}
		f.serve(conn, tlsConfig, implicitTLS, offerSTARTTLS)
	}()
	return f
}

func (f *fakeSMTP) serve(conn net.Conn, tlsConfig *tls.Config, overTLS, offerSTARTTLS bool) {
	tp := textproto.NewConn(conn)
	var s smtpSession
	_ = tp.PrintfLine("220 fake ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		verb := strings.ToUpper(strings.Fields(line + " ")[0])
		switch verb {
		case "EHLO":
			_ = tp.PrintfLine("250-fake")
			if offerSTARTTLS && !overTLS {
				_ = tp.PrintfLine("250-STARTTLS")
			}
			_ = tp.PrintfLine("250 AUTH PLAIN")
		case "STARTTLS":
			_ = tp.PrintfLine("220 ready")
			conn = tls.Server(conn, tlsConfig)
			tp = textproto.NewConn(conn)
			overTLS = true
		case "AUTH":
			s.authOverTLS = overTLS
			_ = tp.PrintfLine("235 ok")
		case "MAIL":
			s.from = line
			_ = tp.PrintfLine("250 ok")
		case "RCPT":
			s.to = line
			_ = tp.PrintfLine("250 ok")
		case "DATA":
			_ = tp.PrintfLine("354 go")
			data, err := tp.ReadDotBytes()
			if err != nil {
				return
			}
			s.data = string(data)
			_ = tp.PrintfLine("250 queued")
		case "QUIT":
			_ = tp.PrintfLine("221 bye")
			f.sessions <- s
			return
		default:
			_ = tp.PrintfLine("502 unsupported")
		}
	}
}

func kindleTestFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.epub")
	if err := os.WriteFile(path, []byte("fake epub"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func kindleTestConfig(port int) KindleConfig {
	return KindleConfig{To: "reader@kindle.com", From: "sender@example.com", SMTPHost: "127.0.0.1", SMTPPort: port}
}

func TestSendToKindleOverImplicitTLS(t *testing.T) {
	tlsConfig := smtpTestTLS(t)
	f := startFakeSMTP(t, tlsConfig, true, false)
	implicitTLSPorts[f.port] = true
	t.Cleanup(func() { delete(implicitTLSPorts, f.port) })

	if err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port), "app-password"); err != nil {
		t.Fatalf("SendToKindle() error = %v", err)
	}
	s := <-f.sessions
	if !s.authOverTLS {
		t.Fatal("credentials were sent before TLS")
	}
	if !strings.Contains(s.from, "sender@example.com") || !strings.Contains(s.to, "reader@kindle.com") {
		t.Fatalf("envelope = %q / %q", s.from, s.to)
	}
	if !strings.Contains(s.data, `filename="book.epub"`) {
		t.Fatalf("message lacks the attachment: %q", s.data)
	}
}

func TestSendToKindleUpgradesWithSTARTTLS(t *testing.T) {
	tlsConfig := smtpTestTLS(t)
	f := startFakeSMTP(t, tlsConfig, false, true)

	if err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port), "app-password"); err != nil {
		t.Fatalf("SendToKindle() error = %v", err)
	}
	if s := <-f.sessions; !s.authOverTLS {
		t.Fatal("credentials were sent before STARTTLS")
	}
}

// A plain client talking to an implicit-TLS server waits for a greeting that
// never comes. That was issue #28; it must now fail fast and name the cause.
func TestSendToKindleTimesOutOnWrongTLSMode(t *testing.T) {
	tlsConfig := smtpTestTLS(t)
	f := startFakeSMTP(t, tlsConfig, true, false)
	original := smtpIdleTimeout
	smtpIdleTimeout = 200 * time.Millisecond
	t.Cleanup(func() { smtpIdleTimeout = original })

	start := time.Now()
	err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port), "app-password")
	if !errors.Is(err, ErrSMTPSend) {
		t.Fatalf("SendToKindle() error = %v, want ErrSMTPSend", err)
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("SendToKindle() error = %v, want a timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("SendToKindle() took %v", elapsed)
	}
	want := net.JoinHostPort("127.0.0.1", strconv.Itoa(f.port)) + " (STARTTLS)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not name %q", err, want)
	}
}

func TestSendToKindleRejectsPlainServerOnImplicitTLSPort(t *testing.T) {
	smtpTestTLS(t)
	f := startFakeSMTP(t, nil, false, false)
	implicitTLSPorts[f.port] = true
	t.Cleanup(func() { delete(implicitTLSPorts, f.port) })

	err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port), "app-password")
	if !errors.Is(err, ErrSMTPSend) || !strings.Contains(err.Error(), "(implicit TLS)") {
		t.Fatalf("SendToKindle() error = %v, want an implicit TLS ErrSMTPSend", err)
	}
	select {
	case <-f.sessions:
		t.Fatal("mail was delivered over a plain connection")
	default:
	}
}
