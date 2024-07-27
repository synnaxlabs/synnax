package skl

import (
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

type Pointer struct {
	telem.TimeRange
	fileKey uint16
	offset  uint32
	length  uint32
}

type Node struct {
	sync.RWMutex
	Ptr  Pointer
	next [MaxHeight]*Node
	prev [MaxHeight]*Node
}

type SkipList struct {
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
	skl.head = &Node{}
	skl.tail = &Node{}
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

// search returns a pointer indicating the largest Node no larger than the requested
// value along with a list of pointers indicating the largest Node no larger than the
// requested value at each level.
// While the first return is simply the first element of the second, it's much more
// convenient to work with.
//
// If the requested Node is smaller than all nodes in the skiplist, skl.head is returned.
// If the requested Node is larger than all nodes in the skiplist, the last element in
// the skiplist is returned.
func (skl *SkipList) search(v telem.TimeRange) (prev *Node, next *Node) {
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

func (skl *SkipList) SearchLE(ts telem.TimeStamp) (*Node, bool) {
	p, _ := skl.search(ts.SpanRange(0))
	if p == skl.head {
		return nil, false
	}
	return p, p.Ptr.ContainsStamp(ts)
}

// SearchGE
func (skl *SkipList) SearchGE(ts telem.TimeStamp) (*Node, bool) {
	p, n := skl.search(ts.SpanRange(0))
	if n == skl.tail {
		return nil, false
	}
	// If the found Node is "equal", then it will be in prev and not next.
	if p.Ptr.ContainsStamp(ts) {
		return p, true
	}
	return n, false
}

func (skl *SkipList) Insert(ptr Pointer) error {
	var (
		prev          = skl.head
		journey       = make([][2]*Node, MaxHeight)
		newNode       = &Node{Ptr: ptr}
		newNodeHeight = randomHeight()
	)
	skl.Lock()
	defer skl.Unlock()

	// Hot Path optimization for inserting to leading edge.
	if skl.checkLeadingEdge(newNode) {
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

	for level := 0; level < newNodeHeight; level++ {
		p, n := journey[level][0], journey[level][1]
		if p == nil {
			newNode.prev[level] = skl.head
			newNode.next[level] = skl.tail
			skl.head.next[level] = newNode
			skl.tail.prev[level] = newNode
			continue
		}
		newNode.prev[level] = p
		newNode.next[level] = n
		p.next[level] = newNode
		n.prev[level] = newNode
	}

	if newNodeHeight > skl.height {
		skl.height = newNodeHeight
	}

	skl.length += 1
	return nil
}

func (skl *SkipList) checkLeadingEdge(newNode *Node) (inserted bool) {
	lastNode := skl.tail.prev[0]

	if newNode.Ptr.Start.AfterEq(lastNode.Ptr.End) {
		// Hot path optimization: appending to end:
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

func (skl *SkipList) Len() int { return skl.length }
