package collision

import (
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestNoCollisions(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 80, End: 80}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 90, End: 90}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions, got %d", len(collisions))
	}
}

func TestSamePortCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "b", Service: "api", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(collisions))
	}
	if len(collisions[0].Bindings) != 2 {
		t.Errorf("expected 2 bindings in collision, got %d", len(collisions[0].Bindings))
	}
}

func TestTCPvsUDPNoCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "dns", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 53, End: 53}},
		{ProjectID: "b", Service: "dns", Protocol: model.ProtocolUDP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 53, End: 53}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions (TCP != UDP), got %d", len(collisions))
	}
}

func TestDifferentHostIPNoCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Specific, Canonical: "192.168.1.1"}, Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Specific, Canonical: "10.0.0.1"}, Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions (different IPs), got %d", len(collisions))
	}
}

func TestWildcardCollidesWithSpecific(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Specific, Canonical: "192.168.1.1"}, Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 1 {
		t.Errorf("expected 1 collision (wildcard vs specific), got %d", len(collisions))
	}
}

func TestRangeOverlapCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8000, End: 8010}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8005, End: 8005}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 1 {
		t.Errorf("expected 1 collision (range overlap), got %d", len(collisions))
	}
}

func TestAdjacentRangesNoCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8000, End: 8004}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8005, End: 8009}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions (adjacent, not overlapping), got %d", len(collisions))
	}
}

func TestSameProjectDuplicate(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 1 {
		t.Errorf("expected 1 collision (same project duplicate), got %d", len(collisions))
	}
}

func TestThreeWayCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "b", Service: "api", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "c", Service: "db", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 1 {
		t.Fatalf("expected 1 collision group, got %d", len(collisions))
	}
	if len(collisions[0].Bindings) != 3 {
		t.Errorf("expected 3 bindings in group, got %d", len(collisions[0].Bindings))
	}
}

func TestIPv4VsIPv6NoCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv6Any}, Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions (IPv4 != IPv6), got %d", len(collisions))
	}
}

func TestOmittedPublishedPortNoCollision(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 0, End: 0}},
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 0, End: 0}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions (ephemeral ports), got %d", len(collisions))
	}
}

func TestStableCollisionID(t *testing.T) {
	bindings := []model.Binding{
		{ProjectID: "b", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
		{ProjectID: "a", Service: "web", Protocol: model.ProtocolTCP, HostIP: model.HostScopeInfo{Scope: model.HostIPv4Any}, Published: model.Interval{Start: 8080, End: 8080}},
	}
	engine := NewEngine(bindings)
	collisions := engine.Detect()
	if len(collisions) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(collisions))
	}
	expectedID := model.CollisionID("collision:tcp:ipv4-any:8080")
	if collisions[0].ID != expectedID {
		t.Errorf("expected collision ID %q, got %q", expectedID, collisions[0].ID)
	}
}
