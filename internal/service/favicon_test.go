package service

import (
	"net"
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

func TestExtractIconCandidates(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/apps/")
	require.NoError(t, err)

	candidates := extractIconCandidates(strings.NewReader(`
		<html>
			<head>
				<link rel="stylesheet" href="/app.css">
				<link rel="apple-touch-icon" href="/apple.png">
				<link rel="shortcut icon" href="favicon.ico">
			</head>
		</html>
	`), baseURL)

	require.Len(t, candidates, 2)
	require.Equal(t, "https://example.com/apple.png", candidates[0].String())
	require.Equal(t, "https://example.com/apps/favicon.ico", candidates[1].String())
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
