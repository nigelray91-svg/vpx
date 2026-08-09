package vaultproxies

import (
	"strconv"
	"strings"
)

// OutputFormats are the line layouts the upstream generator accepts. "host:" is
// accepted in place of "ip:", so both spellings are normalised to the same
// layout here.
var OutputFormats = []string{
	"ip:port:user:pass",
	"user:pass@ip:port",
	"ip:port@user:pass",
	"user:pass:ip:port",
	"ip:port:pass:user",
	"protocol://user:pass@ip:port",
	"protocol://ip:port",
	"ip:port",
	"user:pass",
}

// ValidFormat reports whether f is a supported output format.
func ValidFormat(f string) bool {
	f = normaliseFormat(f)
	for _, v := range OutputFormats {
		if v == f {
			return true
		}
	}
	return false
}

func normaliseFormat(f string) string {
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(f)), "host:", "ip:")
}

// FormatLine renders a single proxy line in the requested layout. It mirrors
// the upstream's own formatting so locally-rendered lists (e.g. re-exporting
// saved credentials) match what the generator returns.
func FormatLine(format, host string, port int, user, pass, protocol string) string {
	hp := host + ":" + strconv.Itoa(port)
	up := user + ":" + pass
	scheme := strings.ToLower(protocol)
	if scheme == "" {
		scheme = "http"
	}

	switch normaliseFormat(format) {
	case "user:pass@ip:port":
		return up + "@" + hp
	case "ip:port@user:pass":
		return hp + "@" + up
	case "user:pass:ip:port":
		return up + ":" + hp
	case "ip:port:pass:user":
		return hp + ":" + pass + ":" + user
	case "protocol://user:pass@ip:port":
		return scheme + "://" + up + "@" + hp
	case "protocol://ip:port":
		return scheme + "://" + hp
	case "ip:port":
		return hp
	case "user:pass":
		return up
	default: // ip:port:user:pass
		return hp + ":" + up
	}
}
