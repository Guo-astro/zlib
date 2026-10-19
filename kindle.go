package zlib

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type KindleConfig struct {
	To       string `json:"to"`
	From     string `json:"from"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
}

const (
	kindleMaxFileSizeBytes     = 200 * 1024 * 1024
	gmailMaxMessageSizeBytes   = 25 * 1024 * 1024
	base64LineLength           = 76
	base64LineBreakLength      = 2
	kindleEstimatedHeaderBytes = 2048
)

func (c KindleConfig) Validate() error {
	if strings.TrimSpace(c.To) == "" {
		return fmt.Errorf("kindle to address is required")
	}
	if strings.TrimSpace(c.From) == "" {
		return fmt.Errorf("kindle from address is required")
	}
	if strings.TrimSpace(c.SMTPHost) == "" {
		return fmt.Errorf("kindle smtp host is required")
	}
	if c.SMTPPort <= 0 {
		return fmt.Errorf("kindle smtp port must be greater than 0")
	}
	return nil
}

func SendToKindle(filePath string, cfg KindleConfig, smtpPassword string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(smtpPassword) == "" {
		return fmt.Errorf("smtp password is required")
	}
	if err := validateKindleAttachment(filePath, cfg); err != nil {
		return err
	}

	msg, err := buildKindleMessage(filePath, cfg)
	if err != nil {
		return err
	}

	auth := smtp.PlainAuth("", cfg.From, smtpPassword, cfg.SMTPHost)
	if err := sendMail(cfg.SMTPHost, cfg.SMTPPort, auth, cfg.From, cfg.To, msg); err != nil {
		mode := "STARTTLS"
		if implicitTLSPorts[cfg.SMTPPort] {
			mode = "implicit TLS"
		}
		addr := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))
		return fmt.Errorf("%w: %s (%s): %w", ErrSMTPSend, addr, mode, err)
	}
	return nil
}

// implicitTLSPorts lists the ports whose servers expect a TLS handshake right
// after connecting (RFC 8314): 465 is SMTP submission over TLS, and 994 is an
// alternative some providers such as smtp.163.com document. Every other port
// connects in plain text and upgrades with STARTTLS (normally 587).
var implicitTLSPorts = map[int]bool{465: true, 994: true}

var (
	smtpDialTimeout = 30 * time.Second
	// smtpIdleTimeout ends a send when the server stops reading or answering.
	// It restarts on every read and write, so a slow upload of a large
	// attachment survives while a silent server does not hang the command.
	smtpIdleTimeout = 2 * time.Minute
	// smtpRootCAs is nil in production (system roots); tests set it.
	smtpRootCAs *x509.CertPool
)

// sendMail delivers msg like smtp.SendMail, but also speaks implicit TLS and
// bounds every network step with a timeout. net/smtp.SendMail only supports
// plain connections upgraded with STARTTLS and has no timeouts, so an
// implicit-TLS server left it waiting for a greeting that never comes.
// smtp.PlainAuth refuses to send credentials before TLS is established.
func sendMail(host string, port int, auth smtp.Auth, from, to string, msg []byte) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	raw, err := (&net.Dialer{Timeout: smtpDialTimeout}).Dial("tcp", addr)
	if err != nil {
		return err
	}
	tlsConfig := &tls.Config{ServerName: host, RootCAs: smtpRootCAs, MinVersion: tls.VersionTLS12}
	conn := net.Conn(&idleTimeoutConn{Conn: raw, timeout: smtpIdleTimeout})
	if implicitTLSPorts[port] {
		conn = tls.Client(conn, tlsConfig)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()

	if !implicitTLSPorts[port] {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConfig); err != nil {
				return err
			}
		}
	}
	if err := c.Auth(auth); err != nil {
		return err
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// idleTimeoutConn pushes the deadline forward before every read and write.
type idleTimeoutConn struct {
	net.Conn
	timeout time.Duration
}

func (c *idleTimeoutConn) Read(b []byte) (int, error) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(c.timeout))
	return c.Conn.Read(b)
}

func (c *idleTimeoutConn) Write(b []byte) (int, error) {
	_ = c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
	return c.Conn.Write(b)
}

func validateKindleAttachment(filePath string, cfg KindleConfig) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".epub", ".pdf", ".txt", ".doc", ".docx", ".html", ".htm", ".rtf", ".jpg", ".jpeg", ".png", ".bmp", ".gif":
	case "":
		return fmt.Errorf("send to kindle failed: file has no extension")
	default:
		return fmt.Errorf("send to kindle failed: unsupported Send-to-Kindle file type %s", ext)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("send to kindle failed: %w", err)
	}
	if info.Size() > kindleMaxFileSizeBytes {
		return fmt.Errorf("send to kindle failed: file size %.2f MB exceeds Amazon's 200 MB limit", float64(info.Size())/1024.0/1024.0)
	}
	if isGmailSMTPHost(cfg.SMTPHost) {
		estimatedMessageSize := estimateBase64MessageSize(info.Size()) + kindleEstimatedHeaderBytes
		if estimatedMessageSize > gmailMaxMessageSizeBytes {
			return fmt.Errorf("send to kindle failed: file size %.2f MB exceeds Gmail's practical 25 MB message limit for attachments", float64(info.Size())/1024.0/1024.0)
		}
	}

	return nil
}

func isGmailSMTPHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	return host == "smtp.gmail.com" || host == "smtp.googlemail.com"
}

func estimateBase64MessageSize(fileSize int64) int64 {
	if fileSize <= 0 {
		return 0
	}
	encodedLen := ((fileSize + 2) / 3) * 4
	lineCount := (encodedLen + base64LineLength - 1) / base64LineLength
	return encodedLen + lineCount*base64LineBreakLength
}

func buildKindleMessage(filePath string, cfg KindleConfig) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(filePath)
	const boundary = "zlib-kindle-boundary"

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", cfg.From)
	fmt.Fprintf(&buf, "To: %s\r\n", cfg.To)
	fmt.Fprintf(&buf, "Subject: Send to Kindle\r\n")
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", boundary)

	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	fmt.Fprintf(&buf, "Sent by zlib-cli.\r\n\r\n")

	attachmentHeader := buildAttachmentHeader(filename)

	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: application/octet-stream; %s\r\n", attachmentHeader)
	fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n")
	fmt.Fprintf(&buf, "Content-Disposition: attachment; %s\r\n\r\n", attachmentHeader)

	encoded := base64.StdEncoding.EncodeToString(data)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		fmt.Fprintf(&buf, "%s\r\n", encoded[i:end])
	}

	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes(), nil
}

func buildAttachmentHeader(filename string) string {
	if isASCII(filename) {
		return fmt.Sprintf("filename=%q", filename)
	}

	fallback := asciiFilenameFallback(filename)
	encoded := url.PathEscape(filename)
	return fmt.Sprintf("filename=%q; filename*=UTF-8''%s", fallback, encoded)
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func asciiFilenameFallback(filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	var b strings.Builder
	for _, r := range base {
		switch {
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		case r <= unicode.MaxASCII && (r == '-' || r == '_' || r == '.'):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte('_')
		default:
			b.WriteByte('_')
		}
	}

	fallback := strings.Trim(b.String(), "._")
	if fallback == "" {
		fallback = "attachment"
	}

	if ext == "" {
		return fallback
	}
	return fallback + ext
}
