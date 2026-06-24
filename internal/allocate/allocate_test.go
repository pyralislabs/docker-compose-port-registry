package allocate

import (
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestAllocateWithRangeBinding(t *testing.T) {
	bindings := []model.Binding{
		{
			ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
		},
		{
			ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8000, End: 8005},
		},
	}
	collisions := []model.Collision{
		{
			ID:        "col1",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				bindings[0],
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Exhausted {
		t.Error("expected successful suggestion")
	}
}

func TestAllocateExhaustedOccupiedSinglePort(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 4000, End: 4000}},
	}
	collisions := []model.Collision{
		{
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4000, bindings)
	results := allocator.Allocate(collisions)

	if !results[0].Exhausted {
		t.Error("expected exhausted (only port 4000 occupied)")
	}
}

func TestAllocateSkipsZeroPublished(t *testing.T) {
	bindings := []model.Binding{
		{
			ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 0, End: 0},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	occupied := allocator.buildOccupiedList()
	if len(occupied) != 0 {
		t.Errorf("expected zero occupied for ephemeral port, got %d", len(occupied))
	}
}

func TestAllocatePreservesOrderForSortedInput(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "api", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80}},
		{ProjectID: "c", Service: "db", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80}},
	}
	collisions := []model.Collision{
		{
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings:  bindings,
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)

	if len(results) != 2 {
		t.Fatalf("expected 2 results (2 losers), got %d", len(results))
	}

	if results[0].Binding.ProjectID != "b" || results[1].Binding.ProjectID != "c" {
		t.Errorf("expected losers in canonical order (b, c), got (%s, %s)",
			results[0].Binding.ProjectID, results[1].Binding.ProjectID)
	}
}

func TestAllocateWildcardBlocksAllSpecifics(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 4000, End: 4000}},
	}
	collisions := []model.Collision{
		{
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "x", Service: "web", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "y", Service: "web", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)

	if results[0].Exhausted {
		t.Error("expected to find candidate (different protocol scope)")
	}
}

func TestSelectWinnerWithEmptyBindings(t *testing.T) {
	allocator := NewAllocator(4000, 4999, nil)
	col := model.Collision{}
	winner := allocator.selectWinner(col)
	if winner.ProjectID != "" {
		t.Errorf("expected empty winner for empty bindings, got %+v", winner)
	}
}

func TestBindingBeforeTieBreak(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Source: model.SourceRef{File: "z.yaml"}},
		{ProjectID: "a", Service: "web", Source: model.SourceRef{File: "a.yaml"}},
	}

	if !bindingBefore(bindings[1], bindings[0]) {
		t.Error("expected bindings[1] (a.yaml) < bindings[0] (z.yaml)")
	}
}

func TestAllocateThreeWayCollision(t *testing.T) {
	bindings := []model.Binding{}
	collisions := []model.Collision{
		{
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "api", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 8080, End: 8080}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 8080, End: 8080}},
				{ProjectID: "c", Service: "db", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 8080, End: 8080}},
			},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)

	if len(results) != 2 {
		t.Fatalf("expected 2 results (2 losers), got %d", len(results))
	}

	if results[0].Suggested.Start != 4000 || results[1].Suggested.Start != 4001 {
		t.Errorf("expected suggestions at 4000, 4001, got %d, %d",
			results[0].Suggested.Start, results[1].Suggested.Start)
	}
}

func TestAllocateDifferentProtocolsIndependent(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "dns", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 4000, End: 4000}},
		{ProjectID: "a", Service: "dns", Protocol: model.ProtocolUDP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 4001, End: 4001}},
	}
	collisions := []model.Collision{
		{
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
					HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
					Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)

	if results[0].Exhausted {
		t.Error("expected suggestion (UDP at 4001 does not block TCP)")
	}
	if results[0].Suggested.Start != 4001 {
		t.Errorf("expected 4001 (TCP at 4000 blocked, UDP at 4001 OK), got %d", results[0].Suggested.Start)
	}
}
