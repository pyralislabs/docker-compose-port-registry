package collision

import (
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestEmptyBindings(t *testing.T) {
	engine := NewEngine(nil)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions for empty input, got %d", len(collisions))
	}
}

func TestSingleBindingNoCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	if len(engine.Detect()) != 0 {
		t.Error("expected no collision with single binding")
	}
}

func TestCollisionWithZeroPublished(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 0, End: 0}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected no collision (one binding has zero port), got %d", len(collisions))
	}
}

func TestCollisionMergedRanges(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8000, End: 8010}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8005, End: 8005}},
		{ProjectID: "c", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8008, End: 8008}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 2 {
		t.Fatalf("expected 2 distinct overlap findings, got %d", len(collisions))
	}
	if len(collisions[0].Bindings) != 2 {
		t.Errorf("expected 2 bindings in first overlap, got %d", len(collisions[0].Bindings))
	}
	if len(collisions[1].Bindings) != 2 {
		t.Errorf("expected 2 bindings in second overlap, got %d", len(collisions[1].Bindings))
	}
}

func TestCollisionsSortedDeterministic(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 9090, End: 9090}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 9090, End: 9090}},
		{ProjectID: "c", Service: "api", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "d", Service: "api", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 2 {
		t.Fatalf("expected 2 collisions, got %d", len(collisions))
	}
	if collisions[0].Published.Start != 8080 {
		t.Errorf("expected first collision at 8080, got %d", collisions[0].Published.Start)
	}
	if collisions[1].Published.Start != 9090 {
		t.Errorf("expected second collision at 9090, got %d", collisions[1].Published.Start)
	}
}

func TestCollisionBindingsSorted(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(collisions))
	}
	if collisions[0].Bindings[0].ProjectID != "a" {
		t.Errorf("expected bindings sorted (a first), got %s", collisions[0].Bindings[0].ProjectID)
	}
}

func TestUnresolvedHostIPNoCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostUnresolved, Address: "myhost"},
			Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions (unresolved host IP excluded), got %d", len(collisions))
	}
}

func TestAddBindingDeduplicates(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP,
			HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
			Published: model.Interval{Start: 8080, End: 8080}},
	}
	result := addBinding(bindings, bindings[0])
	if len(result) != 1 {
		t.Errorf("expected dedup to keep 1 binding, got %d", len(result))
	}

	b2 := model.Binding{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP,
		HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
		Published: model.Interval{Start: 8080, End: 8080}}
	result = addBinding(bindings, b2)
	if len(result) != 2 {
		t.Errorf("expected addBinding to append, got %d", len(result))
	}
}

func TestBindingEqualIgnoresNonIdentityFields(t *testing.T) {
	a := model.Binding{
		ProjectID: "p", Service: "s", Protocol: model.ProtocolTCP,
		HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
		Published: model.Interval{Start: 8080, End: 8080},
		Source:    model.SourceRef{File: "x.yaml"},
	}
	b := a
	b.Target = model.Interval{Start: 9999, End: 9999}
	if !bindingEqual(a, b) {
		t.Error("expected bindingEqual to ignore Target field")
	}
}
