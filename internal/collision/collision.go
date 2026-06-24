package collision

import (
	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

type Engine struct {
	Bindings []model.Binding
}

func NewEngine(bindings []model.Binding) *Engine {
	model.SortBindings(bindings)
	return &Engine{Bindings: bindings}
}

func (e *Engine) Detect() []model.Collision {
	groups := e.groupConflicts()
	var collisions []model.Collision

	for _, group := range groups {
		if len(group.bindings) < 2 {
			continue
		}

		overlapStart := group.overlap.Start
		overlapEnd := group.overlap.End

		overlap := model.Interval{Start: overlapStart, End: overlapEnd}
		collisionID := model.CollisionIDFromParts(group.protocol, group.hostIP, overlap)

		bindings := make([]model.Binding, len(group.bindings))
		copy(bindings, group.bindings)
		model.SortBindings(bindings)

		collisions = append(collisions, model.Collision{
			ID:        collisionID,
			Protocol:  group.protocol,
			HostIP:    group.hostIP,
			Published: overlap,
			Bindings:  bindings,
		})
	}

	model.SortCollisions(collisions)
	return collisions
}

type conflictGroup struct {
	protocol model.Protocol
	hostIP   model.HostScopeInfo
	overlap  model.Interval
	bindings []model.Binding
}

func (e *Engine) groupConflicts() []conflictGroup {
	var groups []conflictGroup

	n := len(e.Bindings)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			bi := e.Bindings[i]
			bj := e.Bindings[j]

			if !protocolsConflict(bi.Protocol, bj.Protocol) {
				continue
			}

			if !model.HostScopesOverlap(bi.HostIP, bj.HostIP) {
				continue
			}

			overlap, ok := bi.Published.Overlap(bj.Published)
			if !ok {
				continue
			}

			if bi.Published.Start == 0 && bi.Published.End == 0 {
				continue
			}
			if bj.Published.Start == 0 && bj.Published.End == 0 {
				continue
			}

			merged := false
			for gi, g := range groups {
				if g.protocol != bi.Protocol {
					continue
				}
				if !model.HostScopesOverlap(g.hostIP, bi.HostIP) {
					continue
				}
				if !g.overlap.Overlaps(overlap) {
					continue
				}
				groups[gi].bindings = addBinding(groups[gi].bindings, bi)
				groups[gi].bindings = addBinding(groups[gi].bindings, bj)
				newOverlap, _ := g.overlap.Overlap(overlap)
				groups[gi].overlap = newOverlap
				merged = true
				break
			}
			if !merged {
				mergedOverlap := overlap
				groups = append(groups, conflictGroup{
					protocol: bi.Protocol,
					hostIP:   bi.HostIP,
					overlap:  mergedOverlap,
					bindings: []model.Binding{bi, bj},
				})
			}
		}
	}

	return groups
}

func protocolsConflict(a, b model.Protocol) bool {
	return a == b
}

func addBinding(bindings []model.Binding, b model.Binding) []model.Binding {
	for _, existing := range bindings {
		if bindingEqual(existing, b) {
			return bindings
		}
	}
	return append(bindings, b)
}

func bindingEqual(a, b model.Binding) bool {
	return a.ProjectID == b.ProjectID &&
		a.Service == b.Service &&
		a.Protocol == b.Protocol &&
		a.HostIP.Canonical == b.HostIP.Canonical &&
		a.Published == b.Published &&
		a.Source.File == b.Source.File
}
