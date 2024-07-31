package domain

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"math"
	"math/rand"
	"sync"
)

// adapted from https://github.com/cloudcentricdev/golang-tutorials/blob/main/03/skiplist/skiplist.go

const (
	MaxHeight = 16
	PValue    = 0.5
)

var probabilities [MaxHeight]uint32

type Node struct {
	// This mutex is locked whenever the node is to be modified. This is necessary even
	// when the entire skiplist is locked.
	sync.RWMutex
	deleted bool
	// invalid is true if the node is on head / tail of the skiplist.
	invalid bool
	Ptr     pointer
	next    [MaxHeight]*Node
	prev    [MaxHeight]*Node
}

type SkipList struct {
	alamos.Instrumentation
	// This mutex is locked whenever an insertion, update, or deletion is to be
	// executed against the entire skiplist.
	sync.RWMutex
	head   *Node
	tail   *Node
	height int
	length int
}

func NewSkipList() *SkipList {
	skl := &SkipList{}
	// Both head and tail are nodes, however, they do not carry a value and are only
	// sentinels.
	skl.head = &Node{invalid: true}
	skl.tail = &Node{invalid: true}
	for level := 0; level < MaxHeight; level++ {
		skl.head.next[level] = skl.tail
		skl.tail.prev[level] = skl.head
	}

	skl.height = 1
	return skl
}

func init() {
	probability := 1.0

	for level := 0; level < MaxHeight; level++ {
		probabilities[level] = uint32(probability * float64(math.MaxUint32))
		probability *= PValue
	}
}

func randomHeight() int {
	seed := rand.Uint32()

	height := 1
	for height < MaxHeight && seed <= probabilities[height] {
		height++
	}

	return height
}

// Search returns a pointer indicating the largest Node no larger than the requested
// value along with a list of pointers indicating the largest Node no larger than the
// requested value at each level.
// While the first return is simply the first element of the second, it's much more
// convenient to work with.
//
// If the requested Node is smaller than all nodes in the skiplist, skl.head is returned.
// If the requested Node is larger than all nodes in the skiplist, the last element in
// the skiplist is returned.
func (skl *SkipList) Search(v telem.TimeRange) (prev *Node, next *Node) {
	skl.RLock()
	skl.RUnlock()
	prev = skl.head

	for level := skl.height - 1; level >= 0; level-- {
		next = prev.next[level]
		for next != skl.tail {
			if next == nil {
				panic("skl reached nil pointer before tail")
			}

			if next.Ptr.Start.AfterEq(v.End) {
				// Move down a level if next value in current level is larger than tr.
				break
			}

			prev = next
			next = next.next[level]
		}
	}

	return prev, next
}

func (skl *SkipList) SearchLE(ctx context.Context, tr telem.TimeRange) (*Node, bool) {
	_, span := skl.T.Bench(ctx, "domain/skl/SearchLE")
	defer span.End()
	p, _ := skl.Search(tr)
	if p == skl.head {
		return nil, false
	}
	return p, p.Ptr.OverlapsWith(tr)
}

func (skl *SkipList) SearchGE(ctx context.Context, tr telem.TimeRange) (*Node, bool) {
	_, span := skl.T.Bench(ctx, "domain/skl/SearchLE")
	defer span.End()
	p, n := skl.Search(tr)
	if n == skl.tail {
		return nil, false
	}
	// If the found Node is "equal", then it will be in prev and not next.
	if p.Ptr.OverlapsWith(tr) {
		return p, true
	}
	return n, false
}

func (skl *SkipList) TimeRange() telem.TimeRange {
	skl.RLock()
	defer skl.RUnlock()
	if skl.length <= 1 {
		return telem.TimeRangeZero
	}
	return skl.head.next[0].Ptr.Start.Range(skl.tail.prev[0].Ptr.End)
}

func (skl *SkipList) Last() *Node {
	skl.RLock()
	defer skl.RUnlock()
	if skl.length == 0 {
		return nil
	}
	return skl.tail.prev[0]
}

func (skl *SkipList) First() *Node {
	skl.RLock()
	defer skl.RUnlock()
	if skl.length == 0 {
		return nil
	}
	return skl.head.next[0]
}

func (skl *SkipList) Insert(ctx context.Context, ptr pointer) error {
	var (
		prev          = skl.head
		journey       = make([][2]*Node, MaxHeight)
		newNode       = &Node{Ptr: ptr}
		newNodeHeight = randomHeight()
	)
	_, span := skl.T.Bench(ctx, "domain/skl/Insert")
	defer span.End()
	skl.Lock()
	defer skl.Unlock()

	// Hot Path optimization for inserting to leading edge.
	if skl.checkLeadingEdgeWrite(newNode) {
		return nil
	}

	for level := skl.height - 1; level >= 0; level-- {
		next := prev.next[level]
		for next != skl.tail {
			if next == nil {
				panic("skl reached nil pointer before tail")
			}

			if next.Ptr.Start.AfterEq(ptr.End) {
				// Move down a level if next value in current level is larger than tr.
				break
			}

			prev = next
			next = next.next[level]
		}
		journey[level] = [2]*Node{prev, next}
		if level == 0 {
			if prev.Ptr.OverlapsWith(ptr.TimeRange) || next.Ptr.OverlapsWith(ptr.TimeRange) {
				return errors.Newf(
					"domain overlap: trying to insert %s between %s and %s",
					ptr.TimeRange,
					prev.Ptr.TimeRange,
					next.Ptr.TimeRange,
				)
			}
		}
	}

	newNode.Lock()
	for level := 0; level < newNodeHeight; level++ {
		p, n := journey[level][0], journey[level][1]
		p.Lock()
		n.Lock()
		newNode.prev[level] = p
		newNode.next[level] = n
		p.next[level] = newNode
		n.prev[level] = newNode
		p.Unlock()
		n.Unlock()
	}
	newNode.Unlock()

	if newNodeHeight > skl.height {
		skl.height = newNodeHeight
	}

	skl.length += 1
	return nil
}

func (skl *SkipList) Update(ctx context.Context, ptr pointer) error {
	_, span := skl.T.Bench(ctx, "domain/skl/Update")
	defer span.End()
	skl.Lock()
	defer skl.Unlock()

	var (
		prev = skl.head
		next = skl.head
	)
	// Hot Path optimization for inserting to leading edge.
	if skl.checkLeadingEdgeUpdate(ptr) {
		return nil
	}

	for level := skl.height - 1; level >= 0; level-- {
		next = prev.next[level]
		for next != skl.tail {
			if next == nil {
				panic("skl reached nil pointer before tail")
			}

			if next.Ptr.Start.AfterEq(ptr.End) {
				// Move down a level if next value in current level is larger than tr.
				break
			}

			prev = next
			next = next.next[level]
		}
	}

	if prev.Ptr.Start != ptr.Start {
		msg := fmt.Sprintf(
			"Index update called with pointer with different start time stamp:"+
				"found start %s in domain, got %s",
			prev.Ptr.Start,
			ptr.Start,
		)
		skl.L.DPanic(msg)
		return span.Error(errors.New(msg))
	}

	if ptr.End.After(next.Ptr.Start) {
		return span.Error(NewErrWriteConflict(ptr.TimeRange, next.Ptr.TimeRange))
	}

	prev.Lock()
	prev.Ptr = ptr
	prev.Unlock()

	return nil
}

func (skl *SkipList) checkLeadingEdgeWrite(newNode *Node) (inserted bool) {
	lastNode := skl.tail.prev[0]

	if newNode.Ptr.Start.AfterEq(lastNode.Ptr.End) {
		height := randomHeight()
		for level := 0; level < height; level++ {
			newNode.prev[level] = lastNode
			newNode.next[level] = skl.tail
			lastNode.next[level] = newNode
			skl.tail.prev[level] = newNode
		}
		if height > skl.height {
			skl.height = height
		}
		skl.length += 1
		return true
	}
	return false
}

func (skl *SkipList) checkLeadingEdgeUpdate(ptr pointer) (updated bool) {
	lastNode := skl.tail.prev[0]

	if lastNode != skl.head && ptr.Start == lastNode.Ptr.Start {
		lastNode.Ptr = ptr
		return true
	}
	return false
}

func (skl *SkipList) Len() int { return skl.length }
