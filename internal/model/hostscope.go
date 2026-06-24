package model

import (
	"fmt"
	"net"
	"strings"
)

func ParseHostScope(s string) HostScopeInfo {
	s = strings.TrimSpace(s)
	if s == "" {
		return HostScopeInfo{Scope: HostAnyUnspecified, Address: "", Canonical: ""}
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return HostScopeInfo{Scope: HostUnresolved, Address: s, Canonical: ""}
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		if ipv4.Equal(net.IPv4zero) {
			return HostScopeInfo{Scope: HostIPv4Any, Address: "0.0.0.0", Canonical: "0.0.0.0"}
		}
		return HostScopeInfo{Scope: HostIPv4Specific, Address: s, Canonical: ipv4.String()}
	}
	if ip.Equal(net.IPv6unspecified) || ip.Equal(net.IPv6zero) {
		return HostScopeInfo{Scope: HostIPv6Any, Address: "::", Canonical: "::"}
	}
	return HostScopeInfo{Scope: HostIPv6Specific, Address: s, Canonical: ip.String()}
}

func HostScopesOverlap(a, b HostScopeInfo) bool {
	if a.Scope == HostUnresolved || b.Scope == HostUnresolved {
		return false
	}
	if a.Scope == HostAnyUnspecified || b.Scope == HostAnyUnspecified {
		return true
	}
	if a.Scope == HostIPv4Any {
		return b.Scope == HostIPv4Any || b.Scope == HostIPv4Specific
	}
	if b.Scope == HostIPv4Any {
		return a.Scope == HostIPv4Any || a.Scope == HostIPv4Specific
	}
	if a.Scope == HostIPv6Any {
		return b.Scope == HostIPv6Any || b.Scope == HostIPv6Specific
	}
	if b.Scope == HostIPv6Any {
		return a.Scope == HostIPv6Any || a.Scope == HostIPv6Specific
	}
	if a.Scope != b.Scope {
		return false
	}
	return a.Canonical == b.Canonical && a.Canonical != ""
}

func CanonicalHostIP(s string) (string, error) {
	info := ParseHostScope(s)
	if info.Scope == HostUnresolved {
		return "", fmt.Errorf("unresolvable host IP: %s", s)
	}
	if info.Scope == HostAnyUnspecified {
		return "", nil
	}
	return info.Canonical, nil
}
