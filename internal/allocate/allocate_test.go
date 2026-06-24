package allocate

import (
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestAllocateEmptyRange(t *testing.T) {
	bindings := []model.Binding{}
	allocator := NewAllocator(4000, 4999, bindings)
	collisions := []model.Collision{}
	results := allocator.Allocate(collisions)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestAllocateSingleCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
	}
	collisions := []model.Collision{
		{
			ID:        "collision:tcp:ipv4-any:80",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Exhausted {
		t.Fatal("unexpected exhausted")
	}
	if results[0].Suggested == nil {
		t.Fatal("expected suggestion")
	}
	if results[0].Suggested.Start != 4000 {
		t.Errorf("expected suggestion start 4000, got %d", results[0].Suggested.Start)
	}
}

func TestAllocateOccupiedLowPorts(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 4000, End: 4000}},
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 4001, End: 4001}},
	}
	collisions := []model.Collision{
		{
			ID:        "collision:tcp:ipv4-any:80",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Suggested == nil || results[0].Suggested.Start != 4002 {
		t.Errorf("expected first gap at 4002, got %d", results[0].Suggested.Start)
	}
}

func TestAllocateDifferentProtocolNoBlock(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "dns", Protocol: model.ProtocolUDP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 4000, End: 4000}},
	}
	collisions := []model.Collision{
		{
			ID:        "collision:tcp:ipv4-any:80",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)
	if results[0].Suggested == nil || results[0].Suggested.Start != 4000 {
		t.Errorf("expected suggestion 4000 (different protocol not blocking), got %d", results[0].Suggested.Start)
	}
}

func TestAllocateRangeWidth(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
	}
	collisions := []model.Collision{
		{
			ID:        "collision:tcp:ipv4-any:80",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)
	if results[0].Suggested == nil {
		t.Fatal("expected suggestion")
	}
	if results[0].Suggested.Width() != 1 {
		t.Errorf("expected width 1, got %d", results[0].Suggested.Width())
	}
}

func TestAllocateExhausted(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 4000, End: 4000}},
	}
	collisions := []model.Collision{
		{
			ID:        "collision:tcp:ipv4-any:80",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4000, 4000, bindings)
	results := allocator.Allocate(collisions)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Exhausted {
		t.Error("expected exhausted, got suggestion")
	}
}

func TestAllocateMultipleCollisions(t *testing.T) {
	bindings := []model.Binding{}
	collisions := []model.Collision{
		{
			ID:        "collision:tcp:ipv4-any:80",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
			},
		},
		{
			ID:        "collision:tcp:ipv4-any:90",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 90, End: 90},
			Bindings: []model.Binding{
				{ProjectID: "c", Service: "api", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 90, End: 90}},
				{ProjectID: "d", Service: "api", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 90, End: 90}},
			},
		},
	}

	allocator := NewAllocator(4000, 4999, bindings)
	results := allocator.Allocate(collisions)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Suggested == nil || results[0].Suggested.Start != 4000 {
		t.Errorf("expected first suggestion at 4000, got %d", results[0].Suggested.Start)
	}
	if results[1].Suggested == nil || results[1].Suggested.Start != 4001 {
		t.Errorf("expected second suggestion at 4001, got %d", results[1].Suggested.Start)
	}
}

func TestAllocateBoundary(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
	}
	collisions := []model.Collision{
		{
			ID:        "collision:tcp:ipv4-any:80",
			Protocol:  model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 80, End: 80},
			Bindings: []model.Binding{
				{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
				{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
			},
		},
	}
	allocator := NewAllocator(4999, 4999, bindings)
	results := allocator.Allocate(collisions)
	if results[0].Exhausted {
		t.Error("expected suggestion at boundary 4999, not exhausted")
	}
	if results[0].Suggested == nil || results[0].Suggested.Start != 4999 {
		t.Errorf("expected suggestion at 4999, got %d", results[0].Suggested.Start)
	}
}
