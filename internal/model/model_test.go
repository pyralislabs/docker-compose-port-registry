package model

import (
	"errors"
	"testing"
)

func TestMutabilityString(t *testing.T) {
	tests := []struct {
		m    Mutability
		want string
	}{
		{Mutable, "mutable"},
		{MutableRange, "refused:range"},
		{MutableInterpolation, "refused:interpolation"},
		{MutableLongSyntax, "refused:long-syntax"},
		{MutableAnchorAlias, "refused:anchor-alias"},
		{MutableOverride, "refused:override"},
		{MutableDuplicate, "refused:duplicate"},
		{MutableReadOnly, "refused:read-only"},
		{Mutability(999), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.m.String(); got != tt.want {
			t.Errorf("Mutability(%d).String() = %q, want %q", tt.m, got, tt.want)
		}
	}
}

func TestCollisionIDString(t *testing.T) {
	c := CollisionID("collision:tcp:ipv4-any:8080")
	if c.String() != "collision:tcp:ipv4-any:8080" {
		t.Errorf("CollisionID.String() unexpected: %s", c.String())
	}
}

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

func TestIntervalFromPort(t *testing.T) {
	i := IntervalFromPort(8080)
	if i.Start != 8080 || i.End != 8080 {
		t.Errorf("expected 8080-8080, got %d-%d", i.Start, i.End)
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

	_, ok = e.Overlap(f)
	if ok {
		t.Error("non-overlapping intervals should not produce overlap")
	}
}

func TestIntervalWidth(t *testing.T) {
	if (Interval{Start: 80, End: 80}).Width() != 1 {
		t.Error("expected width 1 for single port")
	}
	if (Interval{Start: 8000, End: 8005}).Width() != 6 {
		t.Error("expected width 6 for 6-port range")
	}
}

func TestIntervalContains(t *testing.T) {
	a := Interval{Start: 100, End: 200}
	if !a.Contains(Interval{Start: 150, End: 150}) {
		t.Error("larger interval should contain smaller")
	}
	if a.Contains(Interval{Start: 50, End: 150}) {
		t.Error("should not contain overlapping but not contained")
	}
	if !a.Contains(Interval{Start: 100, End: 200}) {
		t.Error("interval should contain itself")
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

func TestSortSourceRefExported(t *testing.T) {
	a := SourceRef{File: "z.yaml"}
	b := SourceRef{File: "a.yaml"}
	if !SortSourceRef(b, a) {
		t.Error("expected a.yaml < z.yaml")
	}
	if SortSourceRef(a, a) {
		t.Error("expected equal not before")
	}
}

func TestHostScopeSortKeyExported(t *testing.T) {
	if got := HostScopeSortKey(HostScopeInfo{Scope: HostIPv4Any, Canonical: "0.0.0.0"}); got != "01-ipv4-any" {
		t.Errorf("unexpected key: %s", got)
	}
}

func TestSortBindingsDeterministic(t *testing.T) {
	bindings := []Binding{
		{ProjectID: "b", Service: "api", Protocol: ProtocolUDP, HostIP: HostScopeInfo{Scope: HostIPv4Any}, Published: Interval{Start: 80, End: 80}, Source: SourceRef{File: "z.yaml"}},
		{ProjectID: "a", Service: "api", Protocol: ProtocolTCP, HostIP: HostScopeInfo{Scope: HostIPv4Any}, Published: Interval{Start: 80, End: 80}, Source: SourceRef{File: "z.yaml"}},
		{ProjectID: "a", Service: "api", Protocol: ProtocolTCP, HostIP: HostScopeInfo{Scope: HostIPv4Any}, Published: Interval{Start: 80, End: 80}, Source: SourceRef{File: "a.yaml"}},
		{ProjectID: "a", Service: "api", Protocol: ProtocolTCP, HostIP: HostScopeInfo{Scope: HostIPv6Any}, Published: Interval{Start: 80, End: 80}},
		{ProjectID: "a", Service: "api", Protocol: ProtocolTCP, HostIP: HostScopeInfo{Scope: HostIPv4Specific, Canonical: "192.168.1.1"}, Published: Interval{Start: 80, End: 80}},
	}
	SortBindings(bindings)

	if bindings[0].ProjectID != "a" || bindings[0].Source.File != "a.yaml" {
		t.Errorf("expected first binding a/a.yaml, got %s/%s", bindings[0].ProjectID, bindings[0].Source.File)
	}
	if bindings[len(bindings)-1].ProjectID != "b" {
		t.Errorf("expected last binding b, got %s", bindings[len(bindings)-1].ProjectID)
	}
}

func TestSortCollisions(t *testing.T) {
	collisions := []Collision{
		{Protocol: ProtocolUDP, HostIP: HostScopeInfo{Scope: HostIPv4Any}, Published: Interval{Start: 53, End: 53}, ID: "z"},
		{Protocol: ProtocolTCP, HostIP: HostScopeInfo{Scope: HostIPv4Any}, Published: Interval{Start: 9090, End: 9090}, ID: "y"},
		{Protocol: ProtocolTCP, HostIP: HostScopeInfo{Scope: HostIPv4Any}, Published: Interval{Start: 8080, End: 8080}, ID: "x"},
		{Protocol: ProtocolTCP, HostIP: HostScopeInfo{Scope: HostIPv6Any}, Published: Interval{Start: 8080, End: 8080}, ID: "w"},
	}
	SortCollisions(collisions)
	if collisions[0].ID != "x" {
		t.Errorf("expected first collision ID x, got %s", collisions[0].ID)
	}
	if collisions[1].ID != "y" {
		t.Errorf("expected second collision ID y, got %s", collisions[1].ID)
	}
}

func TestSortWarnings(t *testing.T) {
	src1 := SourceRef{File: "z.yaml"}
	src2 := SourceRef{File: "a.yaml"}
	warnings := []Warning{
		{Message: "zzz"},
		{Message: "aaa"},
		{Message: "same", Source: &src1},
		{Message: "same", Source: &src2},
		{Message: "same"},
	}
	SortWarnings(warnings)

	if warnings[0].Message != "aaa" {
		t.Errorf("expected first message aaa, got %s", warnings[0].Message)
	}
}

func TestSortFixes(t *testing.T) {
	fixes := []Fix{
		{Binding: Binding{ProjectID: "b", Service: "x", Source: SourceRef{File: "z"}}},
		{Binding: Binding{ProjectID: "a", Service: "z", Source: SourceRef{File: "z"}}},
		{Binding: Binding{ProjectID: "a", Service: "a", Source: SourceRef{File: "z"}}},
		{Binding: Binding{ProjectID: "a", Service: "a", Source: SourceRef{File: "a"}}},
	}
	SortFixes(fixes)

	if fixes[0].Binding.Service != "a" || fixes[0].Binding.Source.File != "a" {
		t.Errorf("expected first fix a/a/a, got %s/%s/%s", fixes[0].Binding.ProjectID, fixes[0].Binding.Service, fixes[0].Binding.Source.File)
	}
}

func TestHostScopeSortKeyAll(t *testing.T) {
	tests := []struct {
		scope HostScopeInfo
		want  string
	}{
		{HostScopeInfo{Scope: HostAnyUnspecified}, "00-any"},
		{HostScopeInfo{Scope: HostIPv4Any}, "01-ipv4-any"},
		{HostScopeInfo{Scope: HostIPv6Any}, "02-ipv6-any"},
		{HostScopeInfo{Scope: HostIPv4Specific, Canonical: "1.2.3.4"}, "10-ipv4:1.2.3.4"},
		{HostScopeInfo{Scope: HostIPv6Specific, Canonical: "::1"}, "11-ipv6:::1"},
		{HostScopeInfo{Scope: HostUnresolved, Address: "host"}, "99-unresolved:host"},
		{HostScopeInfo{Scope: HostScope(99), Address: "x"}, "zz-x"},
	}
	for _, tt := range tests {
		if got := hostScopeSortKey(tt.scope); got != tt.want {
			t.Errorf("hostScopeSortKey(%+v) = %q, want %q", tt.scope, got, tt.want)
		}
	}
}

func TestCollisionIDFromPartsRange(t *testing.T) {
	id := CollisionIDFromParts(ProtocolTCP, HostScopeInfo{Scope: HostIPv6Any, Canonical: "::"}, Interval{Start: 8000, End: 8010})
	want := CollisionID("collision:tcp:ipv6-any:8000-8010")
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}

	idUnresolved := CollisionIDFromParts(ProtocolTCP, HostScopeInfo{Scope: HostUnresolved, Address: "myhost"}, Interval{Start: 80, End: 80})
	wantUnresolved := CollisionID("collision:tcp:unresolved-myhost:80")
	if idUnresolved != wantUnresolved {
		t.Errorf("got %q, want %q", idUnresolved, wantUnresolved)
	}
}

func TestSummaryFromReport(t *testing.T) {
	r := &Report{
		Collisions: []Collision{{}, {}},
		Warnings:   []Warning{{}, {}, {}},
		Fixes: []Fix{
			{Status: FixPlanned},
			{Status: FixPlanned},
			{Status: FixApplied},
			{Status: FixApplied},
			{Status: FixApplied},
			{Status: FixRefused},
		},
	}
	r.Summary = SummaryFromReport(r)

	if r.Summary.Collisions != 2 {
		t.Errorf("expected 2 collisions, got %d", r.Summary.Collisions)
	}
	if r.Summary.Warnings != 3 {
		t.Errorf("expected 3 warnings, got %d", r.Summary.Warnings)
	}
	if r.Summary.FixesPlanned != 2 {
		t.Errorf("expected 2 planned, got %d", r.Summary.FixesPlanned)
	}
	if r.Summary.FixesApplied != 3 {
		t.Errorf("expected 3 applied, got %d", r.Summary.FixesApplied)
	}
}

func TestCountFixesByStatusPublic(t *testing.T) {
	fixes := []Fix{
		{Status: FixApplied},
		{Status: FixApplied},
		{Status: FixPlanned},
		{Status: FixRefused},
	}
	if got := CountFixesByStatus(fixes, FixApplied); got != 2 {
		t.Errorf("expected 2 applied, got %d", got)
	}
	if got := CountFixesByStatus(fixes, FixPlanned); got != 1 {
		t.Errorf("expected 1 planned, got %d", got)
	}
	if got := CountFixesByStatus(fixes, FixRolledBack); got != 0 {
		t.Errorf("expected 0 rolled back, got %d", got)
	}
}

func TestTypedErrorError(t *testing.T) {
	tests := []struct {
		name string
		e    *TypedError
		want string
	}{
		{
			name: "all fields",
			e:    &TypedError{Op: "load", Path: "/x.yaml", Err: errors.New("boom")},
			want: "load: /x.yaml: boom",
		},
		{
			name: "no path",
			e:    &TypedError{Op: "discover", Err: errors.New("nope")},
			want: "discover: nope",
		},
		{
			name: "no err",
			e:    &TypedError{Op: "scan", Path: "/p"},
			want: "scan: /p",
		},
		{
			name: "op only",
			e:    &TypedError{Op: "init"},
			want: "init",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTypedErrorUnwrap(t *testing.T) {
	inner := errors.New("root cause")
	e := &TypedError{Op: "load", Err: inner}
	if e.Unwrap() != inner {
		t.Errorf("Unwrap did not return wrapped error")
	}
	if !errors.Is(e, inner) {
		t.Errorf("errors.Is should find inner error")
	}
}

func TestNewError(t *testing.T) {
	inner := errors.New("inner")
	e := NewError("scan", ErrDiscovery, inner)
	if e.Op != "scan" {
		t.Errorf("expected op scan, got %s", e.Op)
	}
	if e.Type != ErrDiscovery {
		t.Errorf("expected ErrDiscovery, got %v", e.Type)
	}
	if e.Err != inner {
		t.Errorf("expected inner to be wrapped")
	}
}

func TestNewPathError(t *testing.T) {
	inner := errors.New("inner")
	e := NewPathError("load", "/p/compose.yaml", ErrLoad, inner)
	if e.Path != "/p/compose.yaml" {
		t.Errorf("expected path /p/compose.yaml, got %s", e.Path)
	}
	if e.Type != ErrLoad {
		t.Errorf("expected ErrLoad, got %v", e.Type)
	}
}

func TestCanonicalHostIP(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"127.0.0.1", "127.0.0.1", false},
		{"::1", "::1", false},
		{"hostname.example.com", "", true},
	}
	for _, tt := range tests {
		got, err := CanonicalHostIP(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("expected error for %s, got nil", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("unexpected error for %s: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}

	_, err := CanonicalHostIP("")
	if err != nil {
		t.Errorf("empty host IP should not error, got %v", err)
	}
}

func TestHostScopesOverlapAllPaths(t *testing.T) {
	unspecified := HostScopeInfo{Scope: HostAnyUnspecified}
	v4Any := HostScopeInfo{Scope: HostIPv4Any, Canonical: "0.0.0.0"}
	v4Spec := HostScopeInfo{Scope: HostIPv4Specific, Canonical: "192.168.1.1"}
	v6Any := HostScopeInfo{Scope: HostIPv6Any, Canonical: "::"}
	v6Spec := HostScopeInfo{Scope: HostIPv6Specific, Canonical: "::1"}
	unresolved := HostScopeInfo{Scope: HostUnresolved, Address: "x"}

	tests := []struct {
		name string
		a, b HostScopeInfo
		want bool
	}{
		{"unspecified vs v4 any", unspecified, v4Any, true},
		{"unspecified vs v4 spec", unspecified, v4Spec, true},
		{"unspecified vs v6 any", unspecified, v6Any, true},
		{"unspecified vs v6 spec", unspecified, v6Spec, true},
		{"unspecified vs unresolved", unspecified, unresolved, false},
		{"unresolved vs anything", unresolved, v4Any, false},
		{"v4 any vs v4 spec", v4Any, v4Spec, true},
		{"v6 any vs v6 spec", v6Any, v6Spec, true},
		{"v4 spec vs v4 spec same", v4Spec, v4Spec, true},
		{"v6 spec vs v6 spec same", v6Spec, v6Spec, true},
	}
	for _, tt := range tests {
		if got := HostScopesOverlap(tt.a, tt.b); got != tt.want {
			t.Errorf("HostScopesOverlap(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestParseHostScopeIPZeroAddresses(t *testing.T) {
	if info := ParseHostScope("0.0.0.0"); info.Scope != HostIPv4Any {
		t.Errorf("expected HostIPv4Any for 0.0.0.0, got %v", info.Scope)
	}
	if info := ParseHostScope("::"); info.Scope != HostIPv6Any {
		t.Errorf("expected HostIPv6Any for ::, got %v", info.Scope)
	}
}
