package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"navbox/internal/dto"
	"navbox/internal/storage"
)

const (
	faviconFetchTimeout  = 5 * time.Second
	faviconHTMLReadLimit = 256 * 1024
	faviconMaxCandidates = 8
)

func (s *iconService) FetchIcon(ctx context.Context, websiteURL string) (*dto.IconResp, error) {
	baseURL, err := normalizeWebsiteURL(websiteURL)
	if err != nil {
		return nil, err
	}

	fetchCtx, cancel := context.WithTimeout(ctx, faviconFetchTimeout)
	defer cancel()

	client := newRestrictedHTTPClient(s.allowedPrivateCIDRs, s.skipTLSVerify)
	candidates := loadIconCandidates(fetchCtx, client, baseURL)
	for _, candidate := range candidates {
		icon, err := s.fetchAndStoreIcon(fetchCtx, client, candidate)
		if err == nil {
			return icon, nil
		}
		if !errors.Is(err, ErrInvalidInput) && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, ErrNotFound
}

func (s *iconService) fetchAndStoreIcon(ctx context.Context, client *http.Client, iconURL *url.URL) (*dto.IconResp, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iconURL.String(), nil)
	if err != nil {
		return nil, ErrInvalidInput
	}
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/gif,image/x-icon,*/*;q=0.8")
	req.Header.Set("User-Agent", "Navbox/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrNotFound
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, ErrNotFound
	}

	stored, err := s.store.SaveReader(resp.Body)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidIcon) || errors.Is(err, storage.ErrIconTooLarge) {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	return s.saveStoredIcon(ctx, stored)
}

func loadIconCandidates(ctx context.Context, client *http.Client, baseURL *url.URL) []*url.URL {
	candidates := make([]*url.URL, 0, faviconMaxCandidates+1)
	seen := make(map[string]struct{}, faviconMaxCandidates+1)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.String(), nil)
	if err == nil {
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		req.Header.Set("User-Agent", "Navbox/1.0")
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
				content, readErr := io.ReadAll(io.LimitReader(resp.Body, faviconHTMLReadLimit+1))
				if readErr == nil && len(content) <= faviconHTMLReadLimit {
					for _, candidate := range extractIconCandidates(bytes.NewReader(content), resp.Request.URL) {
						candidates = appendIconCandidate(candidates, seen, candidate)
						if len(candidates) >= faviconMaxCandidates {
							break
						}
					}
				}
				baseURL = resp.Request.URL
			}
		}
	}

	fallback := baseURL.ResolveReference(&url.URL{Path: "/favicon.ico"})
	return appendIconCandidate(candidates, seen, fallback)
}

func extractIconCandidates(reader io.Reader, baseURL *url.URL) []*url.URL {
	doc, err := html.Parse(reader)
	if err != nil {
		return nil
	}

	var result []*url.URL
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "link") {
			attrs := htmlAttrs(node)
			if relIncludesIcon(attrs["rel"]) {
				if href := strings.TrimSpace(attrs["href"]); href != "" {
					if parsed, err := url.Parse(href); err == nil {
						result = append(result, baseURL.ResolveReference(parsed))
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return result
}

func htmlAttrs(node *html.Node) map[string]string {
	attrs := make(map[string]string, len(node.Attr))
	for _, attr := range node.Attr {
		attrs[strings.ToLower(attr.Key)] = attr.Val
	}
	return attrs
}

func relIncludesIcon(rel string) bool {
	for _, token := range strings.Fields(strings.ToLower(rel)) {
		if token == "icon" || strings.HasSuffix(token, "-icon") {
			return true
		}
	}
	return false
}

func appendIconCandidate(candidates []*url.URL, seen map[string]struct{}, candidate *url.URL) []*url.URL {
	if validateFetchURL(candidate) != nil {
		return candidates
	}
	key := candidate.String()
	if _, ok := seen[key]; ok {
		return candidates
	}
	seen[key] = struct{}{}
	return append(candidates, candidate)
}

func normalizeWebsiteURL(value string) (*url.URL, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, ErrInvalidInput
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if err := validateFetchURL(parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func validateFetchURL(parsed *url.URL) error {
	if parsed == nil || parsed.Hostname() == "" {
		return ErrInvalidInput
	}
	if parsed.User != nil {
		return ErrInvalidInput
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return ErrInvalidInput
	}
}

func newRestrictedHTTPClient(allowedPrivateCIDRs []netip.Prefix, skipTLSVerify bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if skipTLSVerify {
		// #nosec G402 -- 仅在用户显式开启 icon_fetch.skip_tls_verify 时跳过证书校验。
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	dialer := &net.Dialer{Timeout: faviconFetchTimeout}
	transport.DialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
		return restrictedDialContext(ctx, dialer, network, address, allowedPrivateCIDRs)
	}

	return &http.Client{
		Timeout:   faviconFetchTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return validateFetchURL(req.URL)
		},
	}
}

func restrictedDialContext(ctx context.Context, dialer *net.Dialer, network string, address string, allowedPrivateCIDRs []netip.Prefix) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("split address: %w", err)
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve host: %w", err)
	}
	var dialErr error
	for _, ip := range ips {
		if isAllowedFetchIP(ip.IP, allowedPrivateCIDRs) {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			dialErr = err
		}
	}
	if dialErr != nil {
		return nil, dialErr
	}
	return nil, ErrInvalidInput
}

func isAllowedFetchIP(ip net.IP, allowedPrivateCIDRs []netip.Prefix) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return false
	}
	if addr.IsPrivate() {
		return isAllowedByPrefix(addr, allowedPrivateCIDRs)
	}
	return !isSpecialUseIP(addr)
}

func isAllowedByPrefix(addr netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func isSpecialUseIP(addr netip.Addr) bool {
	if !addr.Is4() {
		return false
	}
	ip := addr.As4()
	switch {
	case ip[0] == 0:
		return true
	case ip[0] == 100 && ip[1]&0xc0 == 64:
		return true
	case ip[0] == 192 && ip[1] == 0 && ip[2] == 0:
		return true
	default:
		return false
	}
}
