package vaultproxies

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// knownGatewayLabels are the fixed gateway labels documented upstream. Each one
// needs a matching CNAME under the brand domain. Anything outside this set is
// almost certainly a per-service gateway (the docs assign resi_unlim_budget a
// hostname per service), which cannot have been pre-created — so the branded
// name would not resolve and the proxy would fail for the customer.
var knownGatewayLabels = map[string]bool{
	"resi-gb": true, "resi": true, "eu-dc": true, "na-dc": true, "dc-gb": true,
	"mobile": true, "na": true, "isp": true, "eu-isp": true, "ipv6": true,
}

// warnedLabels keeps the warning to once per unrecognised label.
var warnedLabels sync.Map

// Brander rewrites upstream gateway hostnames to the operator's own domain.
//
// This is the difference between a whitelabel product and an obvious reseller:
// every generated proxy line embeds the gateway host, so without rewriting,
// customers connect to (and can see) the wholesale provider's DNS.
//
// The operator points their own DNS at the upstream gateways — one CNAME per
// gateway label — and sets PROXY_BRAND_DOMAIN. The proxy protocols here are
// plaintext HTTP/SOCKS5 (no TLS to the gateway itself), so a CNAME resolves to
// the same endpoint and authentication is unaffected.
type Brander struct {
	// explicit maps a full upstream hostname to a branded hostname and takes
	// precedence over domain-suffix rewriting.
	explicit map[string]string
	// domain is the operator's base domain; the upstream's first label is
	// preserved, so resi-gb.vaultproxies.com -> resi-gb.<domain>.
	domain string
	// strict fails instead of emitting an unbranded hostname.
	strict bool
}

// NewBrander builds a Brander. mapSpec is a comma-separated list of
// "upstream.host=branded.host" pairs; either argument may be empty.
func NewBrander(domain, mapSpec string, strict bool) *Brander {
	b := &Brander{
		explicit: map[string]string{},
		domain:   strings.Trim(strings.TrimSpace(strings.ToLower(domain)), "."),
		strict:   strict,
	}
	for _, pair := range strings.Split(mapSpec, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		from, to, ok := strings.Cut(pair, "=")
		from = strings.TrimSpace(strings.ToLower(from))
		to = strings.TrimSpace(to)
		if !ok || from == "" || to == "" {
			continue
		}
		b.explicit[from] = to
	}
	return b
}

// Configured reports whether any branding is in effect.
func (b *Brander) Configured() bool {
	return b != nil && (b.domain != "" || len(b.explicit) > 0)
}

// Host returns the branded hostname for an upstream gateway host.
func (b *Brander) Host(upstream string) (string, error) {
	if upstream == "" {
		return "", nil
	}
	if b == nil || !b.Configured() {
		if b != nil && b.strict {
			return "", fmt.Errorf("proxy hostname branding is required but not configured")
		}
		return upstream, nil
	}
	key := strings.ToLower(strings.TrimSpace(upstream))
	if branded, ok := b.explicit[key]; ok {
		return branded, nil
	}
	if b.domain != "" {
		// Preserve the gateway label so each upstream host keeps a distinct
		// branded name (resi-gb.vaultproxies.com -> resi-gb.<domain>).
		label, _, found := strings.Cut(key, ".")
		if !found || label == "" {
			label = key
		}
		branded := label + "." + b.domain
		// A label we do not recognise has no pre-created CNAME, so the branded
		// hostname will not resolve. Say so loudly rather than handing the
		// customer a dead endpoint.
		if !knownGatewayLabels[label] {
			if _, seen := warnedLabels.LoadOrStore(label, true); !seen {
				slog.Error("unrecognised upstream gateway - branded hostname will not resolve until you add a CNAME",
					"upstream", upstream, "branded", branded,
					"action", "create CNAME "+branded+" -> "+key)
			}
		}
		return branded, nil
	}
	if b.strict {
		return "", fmt.Errorf("no branded hostname configured for upstream gateway %q", upstream)
	}
	return upstream, nil
}

// apply rewrites a generation in place so nothing downstream ever sees the
// upstream hostname — including inside the pre-rendered output line.
func (b *Brander) apply(g *Generation) error {
	branded, err := b.Host(g.Hostname)
	if err != nil {
		return err
	}
	if branded == "" || branded == g.Hostname {
		return nil
	}
	if g.OutputLine != "" {
		// Swap the host inside the upstream's own formatting rather than
		// re-rendering, so the line keeps whatever layout was requested.
		g.OutputLine = strings.ReplaceAll(g.OutputLine, g.Hostname, branded)
	} else {
		g.OutputLine = FormatLine(g.Format, branded, g.Port, g.Username, g.Password, g.Protocol)
	}
	g.Hostname = branded
	return nil
}
