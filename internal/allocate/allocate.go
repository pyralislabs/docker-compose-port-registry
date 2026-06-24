package allocate

import (
	"sort"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

type Allocator struct {
	RangeStart uint16
	RangeEnd   uint16
	Bindings   []model.Binding
}

type AllocationResult struct {
	Binding   model.Binding
	Suggested *model.Interval
	Exhausted bool
}

func NewAllocator(rangeStart, rangeEnd uint16, bindings []model.Binding) *Allocator {
	model.SortBindings(bindings)
	return &Allocator{
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
		Bindings:   bindings,
	}
}

type occupiedInterval struct {
	protocol  model.Protocol
	hostScope model.HostScopeInfo
	interval  model.Interval
}

func (a *Allocator) Allocate(collisions []model.Collision) []AllocationResult {
	occupied := a.buildOccupiedList()
	results := make([]AllocationResult, 0, len(collisions))

	for _, col := range collisions {
		winner := a.selectWinner(col)
		losers := a.selectLosers(col, winner)

		for _, loser := range losers {
			suggestion := a.findCandidate(loser, occupied)
			if suggestion == nil {
				occupied = append(occupied, occupiedInterval{
					protocol:  col.Protocol,
					hostScope: col.HostIP,
					interval:  loser.Published,
				})
				results = append(results, AllocationResult{
					Binding:   loser,
					Exhausted: true,
				})
				continue
			}

			occupied = append(occupied, occupiedInterval{
				protocol:  col.Protocol,
				hostScope: loser.HostIP,
				interval:  *suggestion,
			})

			results = append(results, AllocationResult{
				Binding:   loser,
				Suggested: suggestion,
			})
		}
	}

	return results
}

func (a *Allocator) buildOccupiedList() []occupiedInterval {
	var list []occupiedInterval
	for _, b := range a.Bindings {
		if b.Published.Start == 0 {
			continue
		}
		list = append(list, occupiedInterval{
			protocol:  b.Protocol,
			hostScope: b.HostIP,
			interval:  b.Published,
		})
	}
	return list
}

func (a *Allocator) selectWinner(col model.Collision) model.Binding {
	if len(col.Bindings) == 0 {
		return model.Binding{}
	}
	winner := col.Bindings[0]
	for i := 1; i < len(col.Bindings); i++ {
		if bindingBefore(col.Bindings[i], winner) {
			winner = col.Bindings[i]
		}
	}
	return winner
}

func (a *Allocator) selectLosers(col model.Collision, winner model.Binding) []model.Binding {
	var losers []model.Binding
	for _, b := range col.Bindings {
		if !bindingEqualIdentity(b, winner) {
			losers = append(losers, b)
		}
	}
	sort.Slice(losers, func(i, j int) bool {
		return bindingBefore(losers[i], losers[j])
	})
	return losers
}

func (a *Allocator) findCandidate(binding model.Binding, occupied []occupiedInterval) *model.Interval {
	width := binding.Published.Width()
	for candidate := int(a.RangeStart); candidate+width-1 <= int(a.RangeEnd); candidate++ {
		candInterval := model.Interval{Start: uint16(candidate), End: uint16(candidate + width - 1)}

		if a.isOverlapping(binding, candInterval, occupied) {
			continue
		}

		return &candInterval
	}
	return nil
}

func (a *Allocator) isOverlapping(binding model.Binding, candidate model.Interval, occupied []occupiedInterval) bool {
	for _, occ := range occupied {
		if !protocolsConflict(occ.protocol, binding.Protocol) {
			continue
		}
		if !model.HostScopesOverlap(occ.hostScope, binding.HostIP) {
			continue
		}
		if occ.interval.Overlaps(candidate) {
			return true
		}
	}
	return false
}

func protocolsConflict(a, b model.Protocol) bool {
	return a == b
}

func bindingBefore(a, b model.Binding) bool {
	if a.ProjectID != b.ProjectID {
		return a.ProjectID < b.ProjectID
	}
	if a.Service != b.Service {
		return a.Service < b.Service
	}
	return model.SortSourceRef(a.Source, b.Source)
}

func bindingEqualIdentity(a, b model.Binding) bool {
	return a.ProjectID == b.ProjectID &&
		a.Service == b.Service &&
		a.Source.File == b.Source.File
}
