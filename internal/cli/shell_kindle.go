package cli

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	zlib "github.com/heartleo/zlib"
	"github.com/heartleo/zlib/internal/config"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (m *shellModel) openKindle(arg string) tea.Cmd {
	if arg == "config" {
		cfg, err := config.LoadKindleConfig()
		if err != nil && !os.IsNotExist(err) {
			m.status = err.Error()
			return nil
		}
		if cfg.SMTPHost == "" {
			cfg.SMTPHost = defaultKindleSMTPHost
		}
		if cfg.SMTPPort == 0 {
			cfg.SMTPPort = defaultKindleSMTPPort
		}
		form := newInlineForm("Kindle settings", "Save", []formField{{label: "Kindle address", value: cfg.To}, {label: "From address", value: cfg.From}, {label: "SMTP host", value: cfg.SMTPHost}, {label: "SMTP port", value: strconv.Itoa(cfg.SMTPPort)}})
		form.validate = func(fields []formField) error {
			draft, err := shellKindleConfig(fields)
			if err != nil {
				return err
			}
			return draft.Validate()
		}
		return m.open("kindle-config", form)
	}
	if path, ok := strings.CutPrefix(arg, "send "); ok {
		path = strings.TrimSpace(path)
		if strings.HasPrefix(path, "\"") {
			if !strings.HasSuffix(path, "\"") || len(path) < 2 {
				m.status = "Close the quoted file path"
				return nil
			}
			path = path[1 : len(path)-1]
		}
		if path == "" {
			m.status = "Use /kindle send <file>"
			return nil
		}
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			m.status = "File not found: " + path
			return nil
		}
		m.kindlePath = path
		return m.open("kindle-confirm", shellConfirm{title: "Send to Kindle?", detail: filepath.Base(path), action: "Send"})
	}
	if arg != "" && arg != "send" {
		m.status = "Use /kindle send <file> or /kindle config"
		return nil
	}
	dir, _ := os.Getwd()
	return m.open("kindle-file", newKindleFileModel(dir))
}
func shellKindleConfig(fields []formField) (zlib.KindleConfig, error) {
	port, err := strconv.Atoi(strings.TrimSpace(fields[3].value))
	if err != nil {
		return zlib.KindleConfig{}, fmt.Errorf("invalid SMTP port")
	}
	return zlib.KindleConfig{To: fields[0].value, From: fields[1].value, SMTPHost: fields[2].value, SMTPPort: port}, nil
}
func (m *shellModel) saveKindle(fields []formField) tea.Cmd {
	cfg, err := shellKindleConfig(fields)
	if err == nil {
		err = cfg.Validate()
	}
	if err == nil {
		err = config.SaveKindleConfig(cfg)
	}
	m.restore()
	if err != nil {
		m.status = err.Error()
	} else {
		m.status = "Kindle settings saved"
	}
	return nil
}
func (m *shellModel) sendKindle(path string) tea.Cmd {
	cfg, err := config.LoadKindleConfig()
	if err == nil {
		err = cfg.Validate()
	}
	if err != nil {
		m.status = err.Error()
		m.restore()
		return nil
	}
	password := os.Getenv(kindleSMTPPasswordEnv)
	if password == "" {
		m.status = "Set ZLIB_SMTP_PWD before sending"
		m.restore()
		return nil
	}
	return m.open("kindle-send", newKindleSendModel(path, cfg, password))
}
