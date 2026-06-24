package allocate

import (
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestBindingBeforeByService(t *testing.T) {
	a := model.Binding{ProjectID: "p", Service: "alpha"}
	b := model.Binding{ProjectID: "p", Service: "beta"}

	if !bindingBefore(a, b) {
		t.Error("expected alpha < beta (same project)")
	}
	if bindingBefore(b, a) {
		t.Error("expected beta > alpha")
	}
}

func TestSelectWinnerChoosesMinByService(t *testing.T) {
	allocator := NewAllocator(4000, 4999, nil)
	col := model.Collision{
		Bindings: []model.Binding{
			{ProjectID: "p", Service: "z"},
			{ProjectID: "p", Service: "a"},
		},
	}
	winner := allocator.selectWinner(col)
	if winner.Service != "a" {
		t.Errorf("expected winner service a, got %s", winner.Service)
	}
}

func TestSelectWinnerChoosesMinBySourceFile(t *testing.T) {
	allocator := NewAllocator(4000, 4999, nil)
	col := model.Collision{
		Bindings: []model.Binding{
			{ProjectID: "p", Service: "s", Source: model.SourceRef{File: "z.yaml"}},
			{ProjectID: "p", Service: "s", Source: model.SourceRef{File: "a.yaml"}},
		},
	}
	winner := allocator.selectWinner(col)
	if winner.Source.File != "a.yaml" {
		t.Errorf("expected winner file a.yaml, got %s", winner.Source.File)
	}
}

func TestIsOverlappingDifferentSpecificIPNoOverlap(t *testing.T) {
	allocator := NewAllocator(4000, 4999, nil)
	binding := model.Binding{
		Protocol: model.ProtocolTCP,
		HostIP:   model.HostScopeInfo{Scope: model.HostIPv4Specific, Canonical: "192.168.1.1"},
	}
	occupied := []occupiedInterval{
		{
			protocol:  model.ProtocolTCP,
			hostScope: model.HostScopeInfo{Scope: model.HostIPv4Specific, Canonical: "192.168.1.2"},
			interval:  model.Interval{Start: 4000, End: 4000},
		},
	}
	if allocator.isOverlapping(binding, model.Interval{Start: 4000, End: 4000}, occupied) {
		t.Error("expected no overlap (different specific IPs)")
	}
}

func TestIsOverlappingDifferentProtocolNoOverlap(t *testing.T) {
	allocator := NewAllocator(4000, 4999, nil)
	binding := model.Binding{
		Protocol: model.ProtocolTCP,
		HostIP:   model.HostScopeInfo{Scope: model.HostIPv4Any},
	}
	occupied := []occupiedInterval{
		{
			protocol:  model.ProtocolUDP,
			hostScope: model.HostScopeInfo{Scope: model.HostIPv4Any},
			interval:  model.Interval{Start: 4000, End: 4000},
		},
	}
	if allocator.isOverlapping(binding, model.Interval{Start: 4000, End: 4000}, occupied) {
		t.Error("expected no overlap (different protocols)")
	}
}

func TestIsOverlappingRangeOverlap(t *testing.T) {
	allocator := NewAllocator(4000, 4999, nil)
	binding := model.Binding{
		Protocol: model.ProtocolTCP,
		HostIP:   model.HostScopeInfo{Scope: model.HostIPv4Any},
	}
	occupied := []occupiedInterval{
		{
			protocol:  model.ProtocolTCP,
			hostScope: model.HostScopeInfo{Scope: model.HostIPv4Any},
			interval:  model.Interval{Start: 4005, End: 4010},
		},
	}
	if !allocator.isOverlapping(binding, model.Interval{Start: 4006, End: 4008}, occupied) {
		t.Error("expected overlap (candidate within range)")
	}
	if allocator.isOverlapping(binding, model.Interval{Start: 4011, End: 4015}, occupied) {
		t.Error("expected no overlap (after range)")
	}
	if allocator.isOverlapping(binding, model.Interval{Start: 4000, End: 4004}, occupied) {
		t.Error("expected no overlap (before range)")
	}
}
