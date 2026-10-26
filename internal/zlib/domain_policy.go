package zlib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
	"sync"
)

// allowedDomainSuffixes is the mirror allowlist. Mirrors churn, so it is
// maintained from measurement rather than from the vendor's own domain list:
// several hosts Z-Library advertises are DiamWall-fronted and unusable, and
// several usable ones are not advertised at all. Last measured 2026-09-04:
// z-lib.gd, z-lib.gl, z-library.ec, zlib.bz, article.sk and articles.sk answer
// /eapi/user/login directly; 1lib.sk, z-lib.fm and z-lib.sk self-redirect
// behind DiamWall; zliba.ru is now a plain redirector to zlib.bz.
// 2026-10-25, from the r/zlibrary October list: z-lib.by, z-library.qa and
// z-library.website answer /eapi/info/domains directly; z-library.im and
// z-library.sk self-redirect behind DiamWall but are listed as main domains,
// so they are trusted for when the block lifts. zliba.ru, a redirector to
// zlib.bz that is not on the list, was removed; trust zlib.bz directly.
var allowedDomainSuffixes = []string{
	"1lib.sk",
	"article.sk",
	"articles.sk",
	"z-lib.by",
	"z-lib.fm",
	"z-lib.gd",
	"z-lib.gl",
	"z-lib.sk",
	"z-library.ec",
	"z-library.im",
	"z-library.qa",
	"z-library.sk",
	"z-library.website",
	"zlib.bz",
}

// User mirrors are domains the user has chosen to trust in addition to the
// built-in suffixes: Z-Library invites readers to host their own mirror at any
// address, and donors get personal domains. Each one is trusted as an exact
// host only, never its subdomains, so one mistaken entry cannot widen to a
// whole zone. The CLI installs them with SetUserMirrors at startup.
var (
	userMirrorsMu   sync.RWMutex
	userMirrorHosts []string
)

// AllowedDomainSuffixes returns the built-in upstream hostname suffixes. A
// hostname matches a suffix when it equals it or is one of its subdomains.
func AllowedDomainSuffixes() []string {
	return append([]string(nil), allowedDomainSuffixes...)
}

// ParseAllowedDomain validates an HTTPS origin against the built-in suffixes
// and the user's trusted mirrors, and returns its normalized origin.
func ParseAllowedDomain(raw string) (string, error) {
	target, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || target.Scheme != "https" || target.Host == "" {
		return "", fmt.Errorf("domain must be an HTTPS origin")
	}
	if target.User != nil || target.Port() != "" || target.RawQuery != "" || target.Fragment != "" || (target.Path != "" && target.Path != "/") {
		return "", fmt.Errorf("domain must be an HTTPS origin without credentials, port, path, query, or fragment")
	}

	host := strings.ToLower(strings.TrimSuffix(target.Hostname(), "."))
	if builtInHost(host) || userMirrorHost(host) {
		return "https://" + host, nil
	}
	return "", fmt.Errorf("domain %q is neither a built-in Z-Library domain nor one of your trusted mirrors", host)
}

func builtInHost(host string) bool {
	for _, suffix := range allowedDomainSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func userMirrorHost(host string) bool {
	userMirrorsMu.RLock()
	defer userMirrorsMu.RUnlock()
	return slices.Contains(userMirrorHosts, host)
}

// IsBuiltInDomain reports whether a domain is already covered by the built-in
// suffixes, so adding it as a user mirror would change nothing.
func IsBuiltInDomain(raw string) bool {
	origin, err := ParseMirrorOrigin(raw)
	return err == nil && builtInHost(strings.TrimPrefix(origin, "https://"))
}

// ParseMirrorOrigin normalizes a user mirror to "https://host". A bare host
// such as "mirror.example" is accepted. Plain HTTP, ports, paths, credentials,
// IP addresses and single-label hosts are rejected: the login is sent to this
// host, and each of those is more likely a mistake than a real mirror.
func ParseMirrorOrigin(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("enter a mirror domain such as mirror.example")
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	target, err := url.Parse(value)
	if err != nil || target.Scheme != "https" || target.Host == "" {
		return "", fmt.Errorf("a mirror must be an HTTPS domain")
	}
	if target.User != nil || target.Port() != "" || target.RawQuery != "" || target.Fragment != "" || (target.Path != "" && target.Path != "/") {
		return "", fmt.Errorf("a mirror must be a domain without credentials, port, path, query, or fragment")
	}
	host := strings.ToLower(strings.TrimSuffix(target.Hostname(), "."))
	if net.ParseIP(host) != nil || !strings.Contains(host, ".") {
		return "", fmt.Errorf("a mirror must be a domain name such as mirror.example, not an IP address or a single-label host")
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", fmt.Errorf("%q is not a valid domain name", host)
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return "", fmt.Errorf("%q is not a valid domain name; use its punycode (xn--) form for non-ASCII names", host)
			}
		}
	}
	return "https://" + host, nil
}

// SetUserMirrors replaces the trusted user mirrors. Invalid entries are
// skipped and reported in the returned error; the valid ones still apply.
func SetUserMirrors(origins []string) error {
	hosts := make([]string, 0, len(origins))
	var errs []error
	for _, raw := range origins {
		origin, err := ParseMirrorOrigin(raw)
		if err != nil {
			errs = append(errs, fmt.Errorf("mirror %q: %w", raw, err))
			continue
		}
		host := strings.TrimPrefix(origin, "https://")
		if !slices.Contains(hosts, host) {
			hosts = append(hosts, host)
		}
	}
	userMirrorsMu.Lock()
	userMirrorHosts = hosts
	userMirrorsMu.Unlock()
	return errors.Join(errs...)
}

// UserMirrors returns the trusted user mirrors as origins, in the order added.
func UserMirrors() []string {
	userMirrorsMu.RLock()
	defer userMirrorsMu.RUnlock()
	origins := make([]string, 0, len(userMirrorHosts))
	for _, host := range userMirrorHosts {
		origins = append(origins, "https://"+host)
	}
	return origins
}

// VerifyUserMirror checks, without credentials, that a candidate mirror itself
// answers /eapi/info/domains as the Z-Library EAPI does. A redirect to another
// host fails: the login would go to the first host, so that is the one that
// has to prove it serves Z-Library.
func VerifyUserMirror(ctx context.Context, raw string, opts DomainProbeOptions) (DomainCheck, error) {
	origin, err := ParseMirrorOrigin(raw)
	if err != nil {
		return DomainCheck{}, err
	}
	opts.Path = eapiDomainsInfoPath
	check := ProbeDomainWithOptions(ctx, origin, opts)
	return check, mirrorProbeError(origin, check)
}

// mirrorProbeError judges a /eapi/info/domains probe of origin.
func mirrorProbeError(origin string, check DomainCheck) error {
	if check.Status != DomainStatusHealthy {
		detail := string(check.Status)
		if check.Detail != "" {
			detail += ": " + check.Detail
		}
		return fmt.Errorf("%w: %s (%s)", ErrMirrorUnverified, origin, detail)
	}
	final, err := url.Parse(check.FinalURL)
	if err != nil || final.Scheme != "https" || !strings.EqualFold(final.Hostname(), strings.TrimPrefix(origin, "https://")) {
		return fmt.Errorf("%w: %s redirects to %s; add the domain it redirects to instead", ErrMirrorUnverified, origin, check.FinalURL)
	}
	if !check.zlibEAPI {
		return fmt.Errorf("%w: %s answered, but not with the Z-Library EAPI", ErrMirrorUnverified, origin)
	}
	return nil
}
