// Package netfix repairs DNS resolution on platforms where Go's pure-Go
// resolver has no usable nameserver configuration.
//
// portkey is built with CGO_ENABLED=0, so it always uses Go's pure-Go DNS
// resolver. On Android/Termux there is no /etc/resolv.conf, so the resolver
// falls back to its built-in default of 127.0.0.1:53 / [::1]:53 — addresses
// where nothing is listening, producing "connection refused" on every lookup.
// (Termux's own curl works because it links bionic libc and queries the
// Android system resolver, a path the CGO-less Go binary bypasses.)
package netfix

import (
	"context"
	"net"
	"runtime"
	"time"
)

// fallbackServers are public DNS resolvers tried, in order, when the
// system-configured nameserver cannot be reached.
var fallbackServers = []string{"1.1.1.1:53", "8.8.8.8:53"}

type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// fallbackDial wraps dial so that, when the resolver-provided nameserver fails,
// each fallback server is tried in turn. The original error is returned if all
// fallbacks also fail, keeping error messages meaningful. On a correctly
// configured host the first dial succeeds and no fallback is ever attempted, so
// behaviour there is unchanged.
func fallbackDial(dial dialFunc, fallbacks []string) dialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err == nil {
			return conn, nil
		}
		for _, fb := range fallbacks {
			if c, e := dial(ctx, network, fb); e == nil {
				return c, nil
			}
		}
		return nil, err
	}
}

// Install replaces net.DefaultResolver with one that falls back to public DNS
// servers when the system nameserver is unreachable. It is a no-op on every
// platform except android, where Termux's missing /etc/resolv.conf otherwise
// breaks all name resolution. Call once at startup before any network use.
func Install() {
	if runtime.GOOS != "android" {
		return
	}
	d := &net.Dialer{Timeout: 5 * time.Second}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial:     fallbackDial(d.DialContext, fallbackServers),
	}
}
