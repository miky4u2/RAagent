package common

import (
	"net"
	"net/http"
)

// IsIPAllowed checks if Remote IP is in a slice of allowedIPs
//
func IsIPAllowed(req *http.Request, allowedIPs []string) bool {
	remoteAddress, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return false
	}

	userIP := net.ParseIP(remoteAddress).String()

	if !Find(allowedIPs, userIP) {
		return false
	}
	return true
}
