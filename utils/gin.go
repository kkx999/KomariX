package utils

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func trustedForwardingPeer(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

func forwardedHTTPS(c *gin.Context) bool {
	if !trustedForwardingPeer(c.Request.RemoteAddr) {
		return false
	}
	for _, header := range []string{"X-Forwarded-Proto", "X-Forwarded-Protocol", "X-Url-Scheme"} {
		value := c.Request.Header.Get(header)
		if value == "" {
			continue
		}
		first := strings.TrimSpace(strings.Split(value, ",")[0])
		return strings.EqualFold(first, "https")
	}
	return strings.EqualFold(strings.TrimSpace(c.Request.Header.Get("X-Forwarded-Ssl")), "on")
}

// GetScheme returns https for direct TLS requests or for forwarding headers
// received from a trusted local/private reverse proxy. Forwarded headers from
// arbitrary internet peers are ignored so they cannot downgrade cookie flags
// or poison OAuth callback schemes.
func GetScheme(c *gin.Context) string {
	if c.Request.TLS != nil || forwardedHTTPS(c) {
		return "https"
	}
	return "http"
}

func GetCallbackURL(c *gin.Context) string {
	scheme := GetScheme(c)
	host := c.Request.Host
	return scheme + "://" + host + "/api/oauth_callback"
}
