package service

import (
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeWebsiteURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{name: "with scheme", value: "https://example.com/app", want: "https://example.com/app", ok: true},
		{name: "without scheme", value: "example.com", want: "https://example.com", ok: true},
		{name: "reject ftp", value: "ftp://example.com", ok: false},
		{name: "reject empty", value: "", ok: false},
		{name: "reject user info", value: "https://user@example.com", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeWebsiteURL(tt.value)
			if !tt.ok {
				require.ErrorIs(t, err, ErrInvalidInput)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got.String())
		})
	}
}

func TestExtractPageIconResources(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/apps/")
	require.NoError(t, err)

	resources := extractPageIconResources(strings.NewReader(`
		<html>
			<head>
				<link rel="stylesheet" href="/app.css">
				<link rel="apple-touch-icon" href="/apple.png">
				<link rel="shortcut icon" href="favicon.ico">
				<link rel="manifest" href="/site.webmanifest">
			</head>
		</html>
	`), baseURL)

	require.Len(t, resources.icons, 2)
	require.Equal(t, "https://example.com/apple.png", resources.icons[0].String())
	require.Equal(t, "https://example.com/apps/favicon.ico", resources.icons[1].String())
	require.Len(t, resources.manifests, 1)
	require.Equal(t, "https://example.com/site.webmanifest", resources.manifests[0].String())
}

func TestExtractManifestIconCandidates(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/assets/site.webmanifest")
	require.NoError(t, err)

	candidates := extractManifestIconCandidates([]byte(`{
		"icons": [
			{"src": "icon-192.png"},
			{"src": "/icon-512.png"},
			{"src": ""}
		]
	}`), baseURL)

	require.Len(t, candidates, 2)
	require.Equal(t, "https://example.com/assets/icon-192.png", candidates[0].String())
	require.Equal(t, "https://example.com/icon-512.png", candidates[1].String())
}

func TestNewFaviconRequestBrowserProfile(t *testing.T) {
	targetURL, err := url.Parse("https://example.com/favicon.ico")
	require.NoError(t, err)

	req, err := newFaviconRequest(t.Context(), targetURL, "image/*", faviconRequestProfiles[1])

	require.NoError(t, err)
	require.Equal(t, browserUserAgent, req.Header.Get("User-Agent"))
	require.Equal(t, "en-US,en;q=0.9", req.Header.Get("Accept-Language"))
	require.Equal(t, "image/*", req.Header.Get("Accept"))
}

func TestLoadIconCandidatesIncludesManifestAndCommonPaths(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/apps/home")
	require.NoError(t, err)

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/apps/home":
			return testHTTPResponse(req, http.StatusOK, `<html><head><link rel="icon" href="/page-icon.png"><link rel="manifest" href="/site.webmanifest"></head></html>`), nil
		case "/site.webmanifest":
			return testHTTPResponse(req, http.StatusOK, `{"icons":[{"src":"/manifest-icon.png"}]}`), nil
		default:
			return testHTTPResponse(req, http.StatusNotFound, ""), nil
		}
	})}

	candidates := loadIconCandidates(t.Context(), client, baseURL)
	values := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		values = append(values, candidate.String())
	}

	require.Equal(t, []string{
		"https://example.com/page-icon.png",
		"https://example.com/manifest-icon.png",
		"https://example.com/favicon.ico",
		"https://example.com/favicon.png",
		"https://example.com/apple-touch-icon.png",
		"https://example.com/apple-touch-icon-precomposed.png",
	}, values)
}

func TestLoadPageIconResourcesRetriesWithBrowserProfile(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/")
	require.NoError(t, err)

	var userAgents []string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		userAgents = append(userAgents, req.Header.Get("User-Agent"))
		if req.Header.Get("User-Agent") == navboxUserAgent {
			return testHTTPResponse(req, http.StatusForbidden, ""), nil
		}
		return testHTTPResponse(req, http.StatusOK, `<html><head><link rel="icon" href="/browser-icon.png"></head></html>`), nil
	})}

	resources, finalURL := loadPageIconResources(t.Context(), client, baseURL)

	require.Equal(t, baseURL.String(), finalURL.String())
	require.Len(t, resources.icons, 1)
	require.Equal(t, "https://example.com/browser-icon.png", resources.icons[0].String())
	require.Equal(t, []string{navboxUserAgent, browserUserAgent}, userAgents)
}

func TestLoadIconCandidatesDoesNotRetrySuccessfulPageWithoutDeclaredIcons(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/")
	require.NoError(t, err)

	pageRequests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/" {
			pageRequests++
			return testHTTPResponse(req, http.StatusOK, "<html><head></head></html>"), nil
		}
		return testHTTPResponse(req, http.StatusNotFound, ""), nil
	})}

	candidates := loadIconCandidates(t.Context(), client, baseURL)

	require.Equal(t, 1, pageRequests)
	require.Len(t, candidates, 4)
	require.Equal(t, "https://example.com/favicon.ico", candidates[0].String())
}

func TestIsAllowedFetchIP(t *testing.T) {
	allowedPrivate := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	tests := []struct {
		name    string
		ip      string
		allowed []netip.Prefix
		want    bool
	}{
		{name: "public ipv4", ip: "8.8.8.8", want: true},
		{name: "public ipv6", ip: "2001:4860:4860::8888", want: true},
		{name: "loopback", ip: "127.0.0.1", want: false},
		{name: "private blocked by default", ip: "10.0.0.1", want: false},
		{name: "private allowed by cidr", ip: "10.1.1.99", allowed: allowedPrivate, want: true},
		{name: "link local", ip: "169.254.169.254", want: false},
		{name: "carrier grade nat", ip: "100.64.0.1", want: false},
		{name: "proxy fake ip", ip: "198.18.1.29", want: true},
		{name: "unique local ipv6", ip: "fc00::1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isAllowedFetchIP(net.ParseIP(tt.ip), tt.allowed))
		})
	}
}

func TestNewRestrictedHTTPClientTLSVerifyConfig(t *testing.T) {
	tests := []struct {
		name string
		skip bool
		want bool
	}{
		{name: "default verify tls", skip: false, want: false},
		{name: "skip tls verify", skip: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newRestrictedHTTPClient(nil, tt.skip)
			transport, ok := client.Transport.(*http.Transport)
			require.True(t, ok)
			if tt.want {
				require.NotNil(t, transport.TLSClientConfig)
				require.True(t, transport.TLSClientConfig.InsecureSkipVerify)
				return
			}
			if transport.TLSClientConfig != nil {
				require.False(t, transport.TLSClientConfig.InsecureSkipVerify)
			}
		})
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPResponse(req *http.Request, statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
		Header:     make(http.Header),
	}
}
