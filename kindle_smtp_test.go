package zlib

import (
	"context"
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
	accepted chan struct{}
	stallAt  string
	stalled  chan struct{}
	resume   chan struct{}
}

func (f *fakeSMTP) stall() {
	close(f.stalled)
	<-f.resume
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
	return startFakeSMTPWithStall(t, tlsConfig, implicitTLS, offerSTARTTLS, "")
}

func startFakeSMTPWithStall(t *testing.T, tlsConfig *tls.Config, implicitTLS, offerSTARTTLS bool, stallAt string) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{
		port:     ln.Addr().(*net.TCPAddr).Port,
		sessions: make(chan smtpSession, 1),
		accepted: make(chan struct{}),
		stallAt:  stallAt,
		stalled:  make(chan struct{}),
		resume:   make(chan struct{}),
	}
	done := make(chan struct{})
	connections := make(chan net.Conn, 1)
	t.Cleanup(func() {
		ln.Close()
		close(f.resume)
		select {
		case conn := <-connections:
			conn.Close()
		case <-done:
		}
		<-done
	})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		connections <- conn
		close(f.accepted)
		defer conn.Close()
		if implicitTLS && stallAt == "tls" {
			var hello [1]byte
			if _, err := conn.Read(hello[:]); err == nil {
				f.stall()
			}
			return
		}
		if implicitTLS {
			conn = tls.Server(conn, tlsConfig)
		}
		if stallAt == "greeting" {
			f.stall()
			return
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
		// A TLS ClientHello sent to a plain server is random bytes, so the
		// "line" can be empty; that must not panic the whole test binary.
		verb := ""
		if fields := strings.Fields(line); len(fields) > 0 {
			verb = strings.ToUpper(fields[0])
		}
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
			if f.stallAt == "tls" {
				var hello [1]byte
				if _, err := conn.Read(hello[:]); err == nil {
					f.stall()
				}
				return
			}
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
			if f.stallAt == "upload" {
				if _, err := tp.R.ReadByte(); err == nil {
					f.stall()
				}
				return
			}
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

func TestSendToKindleContextPreCancelledNeverDials(t *testing.T) {
	f := startFakeSMTP(t, nil, false, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// A nonexistent attachment also proves cancellation precedes file loading.
	err := SendToKindleContext(ctx, filepath.Join(t.TempDir(), "missing.epub"), kindleTestConfig(f.port, ""), "test-password")
	if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrSMTPSend) {
		t.Fatalf("SendToKindleContext() error = %v, want context.Canceled and ErrSMTPSend", err)
	}
	if err := sendMailContext(ctx, "127.0.0.1", f.port, false, nil, "", "", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("sendMailContext() error = %v, want context.Canceled", err)
	}
	select {
	case <-f.accepted:
		t.Fatal("cancelled send opened an SMTP connection")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSendToKindleContextCancelsBlockedHandshake(t *testing.T) {
	for _, tt := range []struct {
		name     string
		security string
		stallAt  string
	}{
		{"greeting", SMTPSecuritySTARTTLS, "greeting"},
		{"implicit TLS", SMTPSecurityTLS, "tls"},
		{"STARTTLS", SMTPSecuritySTARTTLS, "tls"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig := smtpTestTLS(t)
			f := startFakeSMTPWithStall(t, tlsConfig, tt.security == SMTPSecurityTLS, true, tt.stallAt)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			path := kindleTestFile(t)
			go func() {
				result <- SendToKindleContext(ctx, path, kindleTestConfig(f.port, tt.security), "test-password")
			}()
			waitForSMTPStall(t, f, result)
			cancel()
			assertSMTPContextError(t, result, context.Canceled)
		})
	}
}

func TestSendToKindleContextDeadlineDuringGreeting(t *testing.T) {
	f := startFakeSMTPWithStall(t, nil, false, true, "greeting")
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	path := kindleTestFile(t)
	result := make(chan error, 1)
	go func() {
		result <- SendToKindleContext(ctx, path, kindleTestConfig(f.port, SMTPSecuritySTARTTLS), "test-password")
	}()
	waitForSMTPStall(t, f, result)
	assertSMTPContextError(t, result, context.DeadlineExceeded)
}

func TestSendToKindleContextCancelsBlockedUpload(t *testing.T) {
	for _, security := range []string{SMTPSecurityTLS, SMTPSecuritySTARTTLS} {
		t.Run(security, func(t *testing.T) {
			tlsConfig := smtpTestTLS(t)
			f := startFakeSMTPWithStall(t, tlsConfig, security == SMTPSecurityTLS, true, "upload")
			path := kindleTestFile(t)
			// Exceed socket buffers so DATA cannot finish while the server stalls.
			if err := os.Truncate(path, 8*1024*1024); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				result <- SendToKindleContext(ctx, path, kindleTestConfig(f.port, security), "test-password")
			}()
			waitForSMTPStall(t, f, result)
			cancel()
			assertSMTPContextError(t, result, context.Canceled)
		})
	}
}

func waitForSMTPStall(t *testing.T, f *fakeSMTP, result <-chan error) {
	t.Helper()
	select {
	case <-f.stalled:
	case err := <-result:
		t.Fatalf("send returned before SMTP stalled: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("SMTP did not reach the blocked phase")
	}
}

func assertSMTPContextError(t *testing.T, result <-chan error, want error) {
	t.Helper()
	select {
	case err := <-result:
		if !errors.Is(err, want) || !errors.Is(err, ErrSMTPSend) {
			t.Fatalf("SendToKindleContext() error = %v, want %v and ErrSMTPSend", err, want)
		}
		if strings.Contains(err.Error(), "set smtp_security") {
			t.Fatalf("cancellation suggested changing TLS mode: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SMTP did not stop promptly after context cancellation")
	}
}
