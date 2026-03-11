package scheduler

import "github.com/sandbox/platform/pkg/types"

// NodeSelector is the interface for node selection algorithms.
type NodeSelector interface {
	Select(nodes []types.NodeInfo) *types.NodeInfo
}

// LeastLoadedSelector picks the node with the lowest current load fraction.
type LeastLoadedSelector struct{}

// Select returns the node with the lowest load. Returns nil if list is empty.
func (s *LeastLoadedSelector) Select(nodes []types.NodeInfo) *types.NodeInfo {
	if len(nodes) == 0 {
		return nil
	}
	best := &nodes[0]
	for i := 1; i < len(nodes); i++ {
		if nodes[i].Load < best.Load {
			best = &nodes[i]
		}
	}
	return best
}
