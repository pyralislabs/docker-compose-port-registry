package model

import (
	"testing"
)

func TestNewInterval(t *testing.T) {
	tests := []struct {
		name   string
		start  uint16
		end    uint16
		wantOk bool
	}{
		{"valid single", 80, 80, true},
		{"valid range", 8000, 8005, true},
		{"invalid zero start", 0, 80, false},
		{"invalid zero end", 80, 0, false},
		{"invalid reversed", 90, 80, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewInterval(tt.start, tt.end)
			if tt.wantOk && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantOk && err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestIntervalOverlap(t *testing.T) {
	a := Interval{Start: 80, End: 80}
	b := Interval{Start: 80, End: 80}
	if !a.Overlaps(b) {
		t.Error("same port should overlap")
	}

	c := Interval{Start: 8000, End: 8010}
	d := Interval{Start: 8005, End: 8005}
	if !c.Overlaps(d) {
		t.Error("range should overlap contained port")
	}

	e := Interval{Start: 80, End: 80}
	f := Interval{Start: 90, End: 90}
	if e.Overlaps(f) {
		t.Error("different ports should not overlap")
	}

	overlap, ok := c.Overlap(d)
	if !ok {
		t.Fatal("expected overlap")
	}
	if overlap.Start != 8005 || overlap.End != 8005 {
		t.Errorf("expected 8005-8005, got %d-%d", overlap.Start, overlap.End)
	}

	g := Interval{Start: 4000, End: 4999}
	h := Interval{Start: 4000, End: 4000}
	if !g.Overlaps(h) {
		t.Error("large range should overlap small")
	}
}

func TestIntervalWidth(t *testing.T) {
	a := Interval{Start: 80, End: 80}
	if a.Width() != 1 {
		t.Errorf("expected width 1, got %d", a.Width())
	}
	b := Interval{Start: 8000, End: 8005}
	if b.Width() != 6 {
		t.Errorf("expected width 6, got %d", b.Width())
	}
}

func TestIntervalContains(t *testing.T) {
	a := Interval{Start: 100, End: 200}
	b := Interval{Start: 150, End: 150}
	if !a.Contains(b) {
		t.Error("larger interval should contain smaller")
	}
	c := Interval{Start: 50, End: 150}
	if a.Contains(c) {
		t.Error("should not contain overlapping but not contained")
	}
}

func TestHostScopeOverlap(t *testing.T) {
	ipv4Any := HostScopeInfo{Scope: HostIPv4Any, Canonical: "0.0.0.0"}
	ipv4Spec := HostScopeInfo{Scope: HostIPv4Specific, Canonical: "192.168.1.1"}
	ipv4Spec2 := HostScopeInfo{Scope: HostIPv4Specific, Canonical: "192.168.1.2"}
	ipv6Any := HostScopeInfo{Scope: HostIPv6Any, Canonical: "::"}
	ipv6Spec := HostScopeInfo{Scope: HostIPv6Specific, Canonical: "::1"}
	anyUnspec := HostScopeInfo{Scope: HostAnyUnspecified}
	unresolved := HostScopeInfo{Scope: HostUnresolved, Address: "some-hostname"}

	tests := []struct {
		name string
		a, b HostScopeInfo
		want bool
	}{
		{"ipv4-wildcard vs specific", ipv4Any, ipv4Spec, true},
		{"ipv4-wildcard vs ipv4-wildcard", ipv4Any, ipv4Any, true},
		{"ipv4-specific vs different", ipv4Spec, ipv4Spec2, false},
		{"ipv4-specific vs same", ipv4Spec, ipv4Spec, true},
		{"ipv4 vs ipv6 wildcard", ipv4Any, ipv6Any, false},
		{"ipv6-wildcard vs specific", ipv6Any, ipv6Spec, true},
		{"any-unspecified vs specific", anyUnspec, ipv4Spec, true},
		{"any-unspecified vs ipv6", anyUnspec, ipv6Any, true},
		{"unresolved vs specific", unresolved, ipv4Spec, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HostScopesOverlap(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("HostScopesOverlap(%+v, %+v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestParseHostScope(t *testing.T) {
	tests := []struct {
		input string
		scope HostScope
	}{
		{"", HostAnyUnspecified},
		{"0.0.0.0", HostIPv4Any},
		{"::", HostIPv6Any},
		{"192.168.1.1", HostIPv4Specific},
		{"::1", HostIPv6Specific},
		{"some-hostname", HostUnresolved},
		{"127.0.0.1", HostIPv4Specific},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			info := ParseHostScope(tt.input)
			if info.Scope != tt.scope {
				t.Errorf("ParseHostScope(%q) = %v, want scope %v", tt.input, info.Scope, tt.scope)
			}
		})
	}
}

func TestCollisionID(t *testing.T) {
	id := CollisionIDFromParts(ProtocolTCP, HostScopeInfo{Scope: HostIPv4Any, Canonical: "0.0.0.0"}, Interval{Start: 8080, End: 8080})
	want := CollisionID("collision:tcp:ipv4-any:8080")
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}

	id2 := CollisionIDFromParts(ProtocolUDP, HostScopeInfo{Scope: HostIPv4Specific, Canonical: "192.168.1.1"}, Interval{Start: 53, End: 53})
	want2 := CollisionID("collision:udp:ipv4-192.168.1.1:53")
	if id2 != want2 {
		t.Errorf("got %q, want %q", id2, want2)
	}
}

func TestSortBindings(t *testing.T) {
	b1 := Binding{ProjectID: "b", Published: Interval{Start: 80, End: 80}}
	b2 := Binding{ProjectID: "a", Published: Interval{Start: 80, End: 80}}
	b3 := Binding{ProjectID: "a", Service: "z", Published: Interval{Start: 90, End: 90}}
	b4 := Binding{ProjectID: "a", Service: "a", Published: Interval{Start: 80, End: 80}}

	bindings := []Binding{b1, b2, b3, b4}
	SortBindings(bindings)

	if bindings[0].ProjectID != "a" || bindings[0].Service != "" {
		t.Errorf("expected first binding to be a/\"\", got %s/%s", bindings[0].ProjectID, bindings[0].Service)
	}
	if bindings[1].ProjectID != "a" || bindings[1].Service != "a" {
		t.Errorf("expected second binding to be a/a, got %s/%s", bindings[1].ProjectID, bindings[1].Service)
	}
	if bindings[3].ProjectID != "b" {
		t.Errorf("expected b last, got %s", bindings[3].ProjectID)
	}
}

func TestIntervalFromPort(t *testing.T) {
	i := IntervalFromPort(8080)
	if i.Start != 8080 || i.End != 8080 {
		t.Errorf("expected 8080-8080, got %d-%d", i.Start, i.End)
	}
}

func TestSortSourceRef(t *testing.T) {
	a := SourceRef{File: "z.yaml"}
	b := SourceRef{File: "a.yaml"}
	if !SortSourceRef(b, a) {
		t.Error("expected a.yaml < z.yaml")
	}
}

func TestHostScopeSortKey(t *testing.T) {
	key := HostScopeSortKey(HostScopeInfo{Scope: HostIPv4Any, Canonical: "0.0.0.0"})
	if key != "01-ipv4-any" {
		t.Errorf("unexpected key: %s", key)
	}
}
