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
	verbs       []string
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
	defer func() { f.sessions <- s }()
	_ = tp.PrintfLine("220 fake ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		verb := strings.ToUpper(strings.Fields(line + " ")[0])
		s.verbs = append(s.verbs, verb)
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

func kindleTestConfig(port int, security string) KindleConfig {
	return KindleConfig{To: "reader@kindle.com", From: "sender@example.com", SMTPHost: "127.0.0.1", SMTPPort: port, SMTPSecurity: security}
}

func TestKindleConfigSecurity(t *testing.T) {
	for _, tt := range []struct {
		port     int
		security string
		want     string
	}{
		{465, "", SMTPSecurityTLS},
		{587, "", SMTPSecuritySTARTTLS},
		{25, "", SMTPSecuritySTARTTLS},
		// 994 is IANA's ircs port; smtp.163.com uses it for implicit TLS, so
		// it needs the explicit setting rather than a port guess.
		{994, "", SMTPSecuritySTARTTLS},
		{994, "tls", SMTPSecurityTLS},
		{465, " STARTTLS ", SMTPSecuritySTARTTLS},
	} {
		if got := kindleTestConfig(tt.port, tt.security).Security(); got != tt.want {
			t.Errorf("Security(port %d, %q) = %q, want %q", tt.port, tt.security, got, tt.want)
		}
	}
	if err := kindleTestConfig(465, "ssl").Validate(); err == nil || !strings.Contains(err.Error(), `"ssl"`) {
		t.Fatalf("Validate() accepted smtp_security ssl: %v", err)
	}
}

func TestSendToKindleOverImplicitTLS(t *testing.T) {
	tlsConfig := smtpTestTLS(t)
	f := startFakeSMTP(t, tlsConfig, true, false)

	if err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port, SMTPSecurityTLS), "app-password"); err != nil {
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

	if err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port, ""), "app-password"); err != nil {
		t.Fatalf("SendToKindle() error = %v", err)
	}
	if s := <-f.sessions; !s.authOverTLS {
		t.Fatal("credentials were sent before STARTTLS")
	}
}

// A server that does not offer STARTTLS, or an attacker who strips it from
// the EHLO reply, must not get the credentials or the book in plain text.
// The loopback address also proves the check does not rely on
// smtp.PlainAuth, which allows plain-text AUTH to localhost.
func TestSendToKindleRefusesMissingSTARTTLS(t *testing.T) {
	smtpTestTLS(t)
	f := startFakeSMTP(t, nil, false, false)

	err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port, ""), "app-password")
	if !errors.Is(err, ErrSMTPSend) || !errors.Is(err, errNoSTARTTLS) {
		t.Fatalf("SendToKindle() error = %v, want errNoSTARTTLS", err)
	}
	s := <-f.sessions
	for _, verb := range s.verbs {
		if verb == "AUTH" || verb == "MAIL" || verb == "DATA" {
			t.Fatalf("client sent %s without TLS; verbs = %v", verb, s.verbs)
		}
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
	err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port, ""), "app-password")
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
	for _, want := range []string{
		net.JoinHostPort("127.0.0.1", strconv.Itoa(f.port)) + " (smtp_security starttls)",
		`set smtp_security to "tls"`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
}

func TestSendToKindleRejectsPlainServerInTLSMode(t *testing.T) {
	smtpTestTLS(t)
	f := startFakeSMTP(t, nil, false, false)

	err := SendToKindle(kindleTestFile(t), kindleTestConfig(f.port, SMTPSecurityTLS), "app-password")
	if !errors.Is(err, ErrSMTPSend) {
		t.Fatalf("SendToKindle() error = %v, want ErrSMTPSend", err)
	}
	for _, want := range []string{"(smtp_security tls)", `set smtp_security to "starttls"`} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
	// The server may log the ClientHello bytes as a garbage line; only real
	// SMTP commands would mean the client fell back to plain text.
	for _, verb := range (<-f.sessions).verbs {
		if verb == "EHLO" || verb == "AUTH" || verb == "MAIL" {
			t.Fatalf("client sent %s in plain text to a TLS-mode port", verb)
		}
	}
}
