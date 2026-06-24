package model

import (
	"fmt"
	"sort"
	"strings"
)

type Interval struct {
	Start uint16 `json:"start"`
	End   uint16 `json:"end"`
}

func NewInterval(start, end uint16) (Interval, error) {
	if start == 0 || end == 0 {
		return Interval{}, fmt.Errorf("port interval values must be in range 1-65535")
	}
	if start > end {
		return Interval{}, fmt.Errorf("interval start %d > end %d", start, end)
	}
	return Interval{Start: start, End: end}, nil
}

func IntervalFromPort(port uint16) Interval {
	return Interval{Start: port, End: port}
}

func (i Interval) Contains(other Interval) bool {
	return i.Start <= other.Start && i.End >= other.End
}

func (i Interval) Overlaps(other Interval) bool {
	return i.Start <= other.End && i.End >= other.Start
}

func (i Interval) Overlap(other Interval) (Interval, bool) {
	if !i.Overlaps(other) {
		return Interval{}, false
	}
	start := i.Start
	if other.Start > start {
		start = other.Start
	}
	end := i.End
	if other.End < end {
		end = other.End
	}
	return Interval{Start: start, End: end}, true
}

func (i Interval) Width() int {
	return int(i.End) - int(i.Start) + 1
}

type Protocol string

const (
	ProtocolTCP Protocol = "tcp"
	ProtocolUDP Protocol = "udp"
)

type HostScope int

const (
	HostAnyUnspecified HostScope = iota
	HostIPv4Any
	HostIPv6Any
	HostIPv4Specific
	HostIPv6Specific
	HostUnresolved
)

type HostScopeInfo struct {
	Scope     HostScope `json:"scope"`
	Address   string    `json:"address,omitempty"`
	Canonical string    `json:"canonical,omitempty"`
}

type SourceRef struct {
	File           string `json:"file"`
	Line           int    `json:"line,omitempty"`
	Column         int    `json:"column,omitempty"`
	OriginalSyntax string `json:"original_syntax,omitempty"`
	OverrideFile   bool   `json:"override_file,omitempty"`
}

type Mutability int

const (
	Mutable              Mutability = iota
	MutableRange                    // range syntax - refused in v1
	MutableInterpolation            // contains ${VAR} - refused in v1
	MutableLongSyntax               // long syntax map - refused in v1
	MutableAnchorAlias              // YAML anchor/alias - refused in v1
	MutableOverride                 // from override file - refused in v1
	MutableDuplicate                // ambiguous duplicate scalar
	MutableReadOnly                 // file is read-only
)

func (m Mutability) String() string {
	switch m {
	case Mutable:
		return "mutable"
	case MutableRange:
		return "refused:range"
	case MutableInterpolation:
		return "refused:interpolation"
	case MutableLongSyntax:
		return "refused:long-syntax"
	case MutableAnchorAlias:
		return "refused:anchor-alias"
	case MutableOverride:
		return "refused:override"
	case MutableDuplicate:
		return "refused:duplicate"
	case MutableReadOnly:
		return "refused:read-only"
	default:
		return "unknown"
	}
}

type Binding struct {
	ProjectID  string        `json:"project_id"`
	Service    string        `json:"service"`
	Protocol   Protocol      `json:"protocol"`
	HostIP     HostScopeInfo `json:"host_ip"`
	Published  Interval      `json:"published"`
	Target     Interval      `json:"target"`
	Source     SourceRef     `json:"source"`
	Mutability Mutability    `json:"mutability"`
}

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Directory string    `json:"directory"`
	Files     []string  `json:"files"`
	Bindings  []Binding `json:"bindings,omitempty"`
}

type CollisionID string

func (c CollisionID) String() string { return string(c) }

type Collision struct {
	ID        CollisionID   `json:"id"`
	Protocol  Protocol      `json:"protocol"`
	HostIP    HostScopeInfo `json:"host_ip"`
	Published Interval      `json:"published"`
	Bindings  []Binding     `json:"bindings"`
}

type Warning struct {
	Message string     `json:"message"`
	Source  *SourceRef `json:"source,omitempty"`
}

type FixStatus int

const (
	FixPlanned    FixStatus = 0
	FixApplied    FixStatus = 1
	FixRefused    FixStatus = 2
	FixRolledBack FixStatus = 3
)

type Fix struct {
	Binding  Binding   `json:"binding"`
	OldValue string    `json:"old_value"`
	NewValue string    `json:"new_value"`
	Status   FixStatus `json:"status"`
	Reason   string    `json:"reason,omitempty"`
}

type Summary struct {
	Projects     int `json:"projects"`
	Bindings     int `json:"bindings"`
	Collisions   int `json:"collisions"`
	Warnings     int `json:"warnings"`
	FixesPlanned int `json:"fixes_planned"`
	FixesApplied int `json:"fixes_applied"`
}

type Report struct {
	SchemaVersion string      `json:"schema_version"`
	ToolVersion   string      `json:"tool_version"`
	Roots         []string    `json:"roots"`
	Summary       Summary     `json:"summary"`
	Collisions    []Collision `json:"collisions,omitempty"`
	Warnings      []Warning   `json:"warnings,omitempty"`
	Fixes         []Fix       `json:"fixes,omitempty"`
}

func SortBindings(bindings []Binding) {
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].ProjectID != bindings[j].ProjectID {
			return bindings[i].ProjectID < bindings[j].ProjectID
		}
		if bindings[i].Service != bindings[j].Service {
			return bindings[i].Service < bindings[j].Service
		}
		if bindings[i].Protocol != bindings[j].Protocol {
			return string(bindings[i].Protocol) < string(bindings[j].Protocol)
		}
		hi := bindings[i].HostIP
		hj := bindings[j].HostIP
		if hi.Scope != hj.Scope {
			return hi.Scope < hj.Scope
		}
		if hi.Address != hj.Address {
			return hi.Address < hj.Address
		}
		if bindings[i].Published.Start != bindings[j].Published.Start {
			return bindings[i].Published.Start < bindings[j].Published.Start
		}
		if bindings[i].Published.End != bindings[j].Published.End {
			return bindings[i].Published.End < bindings[j].Published.End
		}
		return sortSourceRef(bindings[i].Source, bindings[j].Source)
	})
}

func sortSourceRef(a, b SourceRef) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Column < b.Column
}

func SortCollisions(collisions []Collision) {
	sort.Slice(collisions, func(i, j int) bool {
		if string(collisions[i].Protocol) != string(collisions[j].Protocol) {
			return string(collisions[i].Protocol) < string(collisions[j].Protocol)
		}
		hsi := hostScopeSortKey(collisions[i].HostIP)
		hsj := hostScopeSortKey(collisions[j].HostIP)
		if hsi != hsj {
			return hsi < hsj
		}
		if collisions[i].Published.Start != collisions[j].Published.Start {
			return collisions[i].Published.Start < collisions[j].Published.Start
		}
		return collisions[i].Published.End < collisions[j].Published.End
	})
}

func SortWarnings(warnings []Warning) {
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Message != warnings[j].Message {
			return warnings[i].Message < warnings[j].Message
		}
		if warnings[i].Source != nil && warnings[j].Source != nil {
			return sortSourceRef(*warnings[i].Source, *warnings[j].Source)
		}
		return warnings[i].Source == nil
	})
}

func SortFixes(fixes []Fix) {
	sort.Slice(fixes, func(i, j int) bool {
		return sortBindingRef(fixes[i].Binding, fixes[j].Binding)
	})
}

func sortBindingRef(a, b Binding) bool {
	if a.ProjectID != b.ProjectID {
		return a.ProjectID < b.ProjectID
	}
	if a.Service != b.Service {
		return a.Service < b.Service
	}
	return sortSourceRef(a.Source, b.Source)
}

func hostScopeSortKey(h HostScopeInfo) string {
	switch h.Scope {
	case HostAnyUnspecified:
		return "00-any"
	case HostIPv4Any:
		return "01-ipv4-any"
	case HostIPv6Any:
		return "02-ipv6-any"
	case HostIPv4Specific:
		return "10-ipv4:" + h.Canonical
	case HostIPv6Specific:
		return "11-ipv6:" + h.Canonical
	case HostUnresolved:
		return "99-unresolved:" + h.Address
	default:
		return "zz-" + h.Address
	}
}

func CollisionIDFromParts(protocol Protocol, hostIP HostScopeInfo, published Interval) CollisionID {
	ipPart := "any"
	switch hostIP.Scope {
	case HostIPv4Any:
		ipPart = "ipv4-any"
	case HostIPv6Any:
		ipPart = "ipv6-any"
	case HostIPv4Specific:
		ipPart = "ipv4-" + hostIP.Canonical
	case HostIPv6Specific:
		ipPart = "ipv6-" + hostIP.Canonical
	case HostUnresolved:
		ipPart = "unresolved-" + hostIP.Address
	}
	startStr := fmt.Sprintf("%d", published.Start)
	if published.Start == published.End {
		return CollisionID(fmt.Sprintf("collision:%s:%s:%s", protocol, ipPart, startStr))
	}
	return CollisionID(fmt.Sprintf("collision:%s:%s:%d-%d", protocol, ipPart, published.Start, published.End))
}

func SummaryFromReport(r *Report) Summary {
	return Summary{
		Projects:     r.Summary.Projects,
		Bindings:     r.Summary.Bindings,
		Collisions:   len(r.Collisions),
		Warnings:     len(r.Warnings),
		FixesPlanned: countFixesByStatus(r.Fixes, FixPlanned),
		FixesApplied: countFixesByStatus(r.Fixes, FixApplied),
	}
}

func countFixesByStatus(fixes []Fix, status FixStatus) int {
	count := 0
	for _, f := range fixes {
		if f.Status == status {
			count++
		}
	}
	return count
}

type ReportFilterOptions struct {
	RelativePaths bool
}

type TypedError struct {
	Op   string
	Path string
	Type ErrorType
	Err  error
}

type ErrorType int

const (
	ErrInvalidConfig ErrorType = iota
	ErrDiscovery
	ErrLoad
	ErrUnsupported
	ErrIndeterminate
	ErrCollision
	ErrAllocationExhausted
	ErrFixRefused
	ErrTransaction
	ErrInternal
)

func (e *TypedError) Error() string {
	var b strings.Builder
	b.WriteString(e.Op)
	if e.Path != "" {
		b.WriteString(": ")
		b.WriteString(e.Path)
	}
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}
	return b.String()
}

func (e *TypedError) Unwrap() error { return e.Err }

func NewError(op string, errType ErrorType, err error) *TypedError {
	return &TypedError{Op: op, Type: errType, Err: err}
}

func NewPathError(op, path string, errType ErrorType, err error) *TypedError {
	return &TypedError{Op: op, Path: path, Type: errType, Err: err}
}
