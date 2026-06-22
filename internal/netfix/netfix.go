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
	"errors"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// getpropPaths are the locations getprop may live at; Termux's PATH does not
// always include /system/bin, so the absolute path is tried as a fallback.
var getpropPaths = []string{"getprop", "/system/bin/getprop"}

// publicDNS are last-resort resolvers, used only when the Android system
// properties yield no usable nameserver.
var publicDNS = []string{"1.1.1.1:53", "8.8.8.8:53"}

type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// getprop returns the value of an Android system property, or "" if it can't
// be read (getprop missing, non-Android, empty value).
func getprop(name string) string {
	for _, bin := range getpropPaths {
		out, err := exec.Command(bin, name).Output()
		if err != nil {
			continue
		}
		return strings.TrimSpace(string(out))
	}
	return ""
}

// realDNSServers returns the nameservers to use in place of the dead loopback
// defaults. The Android network's own resolvers (net.dnsN system properties)
// come first — they work even on networks that block outbound DNS to public
// resolvers — followed by public DNS as a fallback.
func realDNSServers() []string {
	var servers []string
	for _, prop := range []string{"net.dns1", "net.dns2"} {
		ip := getprop(prop)
		if ip == "" || net.ParseIP(ip) == nil {
			continue
		}
		servers = append(servers, net.JoinHostPort(ip, "53"))
	}
	return append(servers, publicDNS...)
}

// isLoopbackHostPort reports whether address is a loopback nameserver such as
// the 127.0.0.1:53 / [::1]:53 that Go's pure resolver falls back to.
func isLoopbackHostPort(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// redirectDial wraps dial so that requests to a loopback nameserver are sent to
// a real nameserver instead. Dials to any other address pass through unchanged,
// so a host with a valid resolv.conf is unaffected.
//
// The redirect is required rather than a fallback-on-error: a UDP dial to a
// dead local port SUCCEEDS (UDP is connectionless, so nothing is sent at dial
// time) and the "connection refused" only surfaces on a later read, which a
// Dial hook cannot intercept. So the loopback socket must never be handed back.
func redirectDial(dial dialFunc, servers []string) dialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if !isLoopbackHostPort(address) {
			return dial(ctx, network, address)
		}
		var lastErr error
		for _, s := range servers {
			conn, err := dial(ctx, network, s)
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = errors.New("netfix: no usable nameserver")
		}
		return nil, lastErr
	}
}

// Install replaces net.DefaultResolver with one that redirects the dead
// loopback nameserver to a real one. It is a no-op on every platform except
// android, where Termux's missing /etc/resolv.conf otherwise breaks all name
// resolution. Call once at startup before any network use.
func Install() {
	if runtime.GOOS != "android" {
		return
	}
	d := &net.Dialer{Timeout: 5 * time.Second}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial:     redirectDial(d.DialContext, realDNSServers()),
	}
}
