package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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
	faviconFetchTimeout          = 5 * time.Second
	faviconHTMLReadLimit         = 256 * 1024
	faviconManifestReadLimit     = 64 * 1024
	faviconMaxPageCandidates     = 8
	faviconMaxManifestCandidates = 4
	faviconMaxManifests          = 2
	faviconMaxCandidates         = 16
	navboxUserAgent              = "Navbox/1.0"
	browserUserAgent             = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

type faviconRequestProfile struct {
	userAgent      string
	acceptLanguage string
}

type pageIconResources struct {
	icons     []*url.URL
	manifests []*url.URL
}

type webManifest struct {
	Icons []struct {
		Src string `json:"src"`
	} `json:"icons"`
}

var faviconRequestProfiles = []faviconRequestProfile{
	{userAgent: navboxUserAgent},
	{userAgent: browserUserAgent, acceptLanguage: "en-US,en;q=0.9"},
}

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
	var lastErr error
	for _, profile := range faviconRequestProfiles {
		req, err := newFaviconRequest(ctx, iconURL, "image/avif,image/webp,image/png,image/jpeg,image/gif,image/x-icon,*/*;q=0.8", profile)
		if err != nil {
			return nil, ErrInvalidInput
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = ErrNotFound
			continue
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			_ = resp.Body.Close()
			lastErr = ErrNotFound
			continue
		}

		stored, err := s.store.SaveReader(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			if errors.Is(err, storage.ErrInvalidIcon) || errors.Is(err, storage.ErrIconTooLarge) {
				lastErr = ErrInvalidInput
				continue
			}
			return nil, err
		}
		return s.saveStoredIcon(ctx, stored)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrNotFound
}

func loadIconCandidates(ctx context.Context, client *http.Client, baseURL *url.URL) []*url.URL {
	candidates := make([]*url.URL, 0, faviconMaxCandidates)
	seen := make(map[string]struct{}, faviconMaxCandidates)

	resources, finalURL := loadPageIconResources(ctx, client, baseURL)
	baseURL = finalURL
	for _, candidate := range resources.icons {
		candidates = appendIconCandidate(candidates, seen, candidate)
		if len(candidates) >= faviconMaxPageCandidates {
			break
		}
	}

	manifestCandidates := 0
	for index, manifestURL := range resources.manifests {
		if index >= faviconMaxManifests || manifestCandidates >= faviconMaxManifestCandidates {
			break
		}
		if validateFetchURL(manifestURL) != nil {
			continue
		}
		for _, candidate := range loadManifestIconCandidates(ctx, client, manifestURL) {
			before := len(candidates)
			candidates = appendIconCandidate(candidates, seen, candidate)
			if len(candidates) > before {
				manifestCandidates++
			}
			if manifestCandidates >= faviconMaxManifestCandidates {
				break
			}
		}
	}

	for _, path := range []string{"/favicon.ico", "/favicon.png", "/apple-touch-icon.png", "/apple-touch-icon-precomposed.png"} {
		candidates = appendIconCandidate(candidates, seen, baseURL.ResolveReference(&url.URL{Path: path}))
	}
	if len(candidates) > faviconMaxCandidates {
		return candidates[:faviconMaxCandidates]
	}
	return candidates
}

func loadPageIconResources(ctx context.Context, client *http.Client, baseURL *url.URL) (pageIconResources, *url.URL) {
	finalURL := baseURL
	for _, profile := range faviconRequestProfiles {
		req, err := newFaviconRequest(ctx, baseURL, "text/html,application/xhtml+xml", profile)
		if err != nil {
			return pageIconResources{}, finalURL
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			_ = resp.Body.Close()
			continue
		}
		content, readErr := io.ReadAll(io.LimitReader(resp.Body, faviconHTMLReadLimit+1))
		_ = resp.Body.Close()
		if resp.Request != nil && resp.Request.URL != nil {
			finalURL = resp.Request.URL
		}
		if readErr != nil || len(content) > faviconHTMLReadLimit {
			return pageIconResources{}, finalURL
		}
		resources := extractPageIconResources(bytes.NewReader(content), finalURL)
		return resources, finalURL
	}
	return pageIconResources{}, finalURL
}

func loadManifestIconCandidates(ctx context.Context, client *http.Client, manifestURL *url.URL) []*url.URL {
	for _, profile := range faviconRequestProfiles {
		req, err := newFaviconRequest(ctx, manifestURL, "application/manifest+json,application/json", profile)
		if err != nil {
			return nil
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			_ = resp.Body.Close()
			continue
		}
		content, readErr := io.ReadAll(io.LimitReader(resp.Body, faviconManifestReadLimit+1))
		_ = resp.Body.Close()
		if readErr != nil || len(content) > faviconManifestReadLimit {
			return nil
		}
		baseURL := manifestURL
		if resp.Request != nil && resp.Request.URL != nil {
			baseURL = resp.Request.URL
		}
		return extractManifestIconCandidates(content, baseURL)
	}
	return nil
}

func extractPageIconResources(reader io.Reader, baseURL *url.URL) pageIconResources {
	doc, err := html.Parse(reader)
	if err != nil {
		return pageIconResources{}
	}

	result := pageIconResources{}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "link") {
			attrs := htmlAttrs(node)
			if href := strings.TrimSpace(attrs["href"]); href != "" {
				if parsed, err := url.Parse(href); err == nil {
					switch {
					case relIncludesIcon(attrs["rel"]):
						result.icons = append(result.icons, baseURL.ResolveReference(parsed))
					case relIncludesToken(attrs["rel"], "manifest"):
						result.manifests = append(result.manifests, baseURL.ResolveReference(parsed))
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

func extractManifestIconCandidates(content []byte, baseURL *url.URL) []*url.URL {
	var manifest webManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return nil
	}

	result := make([]*url.URL, 0, len(manifest.Icons))
	for _, icon := range manifest.Icons {
		src := strings.TrimSpace(icon.Src)
		if src == "" {
			continue
		}
		parsed, err := url.Parse(src)
		if err != nil {
			continue
		}
		result = append(result, baseURL.ResolveReference(parsed))
	}
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

func relIncludesToken(rel string, target string) bool {
	for _, token := range strings.Fields(strings.ToLower(rel)) {
		if token == target {
			return true
		}
	}
	return false
}

func newFaviconRequest(ctx context.Context, targetURL *url.URL, accept string, profile faviconRequestProfile) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", profile.userAgent)
	if profile.acceptLanguage != "" {
		req.Header.Set("Accept-Language", profile.acceptLanguage)
	}
	return req, nil
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
