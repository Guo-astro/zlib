package zlib

import (
	"os"
	"strings"
)

const (
	DefaultDomain = "https://z-lib.gd"
	TorDomain     = "http://bookszlibb74ugqojhzhg2a63w5i2atv5bqarulgczawnbmsb6s6qead.onion"
	EnvDomain     = "ZLIB_DOMAIN"
	EnvProxy      = "ZLIB_PROXY"
	EnvRetries    = "ZLIB_DOWNLOAD_RETRIES"
)

// defaultDomainOverride is set only by SetDefaultDomain. Empty means "not
// overridden", which lets CurrentDefaultDomain fall through to the environment.
var defaultDomainOverride string

// CurrentDefaultDomain resolves the domain new clients start from:
// SetDefaultDomain > ZLIB_DOMAIN > DefaultDomain.
//
// ZLIB_DOMAIN is read on every call rather than in an init function: the CLI
// loads .env after all init functions have run, so an init-time read would
// never observe a .env-supplied value.
func CurrentDefaultDomain() string {
	if defaultDomainOverride != "" {
		return defaultDomainOverride
	}
	if domain := normalizeDomain(os.Getenv(EnvDomain)); domain != "" {
		return domain
	}
	return DefaultDomain
}

// SetDefaultDomain overrides the default domain for clients created afterwards.
// Passing an empty string clears the override, restoring ZLIB_DOMAIN-or-default
// resolution.
func SetDefaultDomain(domain string) {
	defaultDomainOverride = normalizeDomain(domain)
}

func normalizeDomain(domain string) string {
	return strings.TrimRight(strings.TrimSpace(domain), "/")
}

func absolutizeURL(base, href string) string {
	if href == "" {
		return ""
	}
	if isAbsoluteURL(href) {
		return href
	}
	return base + href
}

func isAbsoluteURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")
}
