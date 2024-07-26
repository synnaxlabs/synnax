package skl

import (
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

type node struct {
	sync.RWMutex
	ptr  Pointer
	next [MaxHeight]*node
	prev [MaxHeight]*node
}

type SkipList struct {
	head   *node
	tail   *node
	height int
	length int
}

func NewSkipList() *SkipList {
	skl := &SkipList{}
	// Both head and tail are nodes, however, they do not carry a value and are only
	// sentinels.
	skl.head = &node{}
	skl.tail = &node{}
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

// search returns a pointer indicating the largest node no larger than the requested
// value along with a list of pointers indicating the largest node no larger than the
// requested value at each level.
// While the first return is simply the first element of the second, it's much more
// convenient to work with.
//
// If the requested node is smaller than all nodes in the skiplist, skl.head is returned.
// If the requested node is larger than all nodes in the skiplist, the last element in
// the skiplist is returned.
func (skl *SkipList) search(v telem.TimeRange) (prev *node, next *node) {
	prev = skl.head
	prev.RLock()

	for level := skl.height - 1; level >= 0; level-- {
		next = prev.next[level]
		next.RLock()
		for next != skl.tail {
			if next == nil {
				panic("skl reached nil pointer before tail")
			}

			if next.ptr.Start.Before(v.End) {
				// Move down a level if next value in current level is larger than tr.
				break
			}

			prev.RUnlock()
			prev = next
			next = next.next[level]
			next.RLock()
		}
		next.RUnlock()
	}
	prev.RUnlock()

	return prev, next
}

func (skl *SkipList) SearchLE(ts telem.TimeStamp) (Pointer, bool) {
	p, _ := skl.search(ts.SpanRange(0))
	if p == skl.head {
		return Pointer{}, false
	}
	return p.ptr, true
}

func (skl *SkipList) SearchGE(ts telem.TimeStamp) (Pointer, bool) {
	_, n := skl.search(ts.SpanRange(0))
	if n == skl.tail {
		return Pointer{}, false
	}
	return n.ptr, true
}

func (skl *SkipList) Insert(ptr Pointer) {
	var (
		prev        = skl.head
		nodesLocked = make([]*node, 0)
	)
	prev.RLock()

	for level := skl.height - 1; level >= 0; level-- {
		next := prev.next[level]
		next.RLock()
		for next != skl.tail {
			if next == nil {
				panic("skl reached nil pointer before tail")
			}

			if {
				// Move down a level if next value in current level is larger than tr.
				break
			}

			prev.RUnlock()
			prev = next
			next = next.next[level]
			next.RLock()
		}
		nodesLocked = append(nodesLocked, []*node{prev, next}...)
	}
	prev.RUnlock()

	if height > skl.height {
		skl.height = height
	}

	skl.length += 1
}

func (skl *SkipList) Delete(start int, end int) {
	target, _ := skl.search(start)
	if target.tr != start {
		target = target.next[0]
	}

	for target.tr < end && target != skl.tail {
		for level := 0; level < skl.height; level++ {
			if target.prev[level] != nil {
				target.next[level].prev[level] = target.prev[level]
				target.prev[level].next[level] = target.next[level]
			}
		}
		skl.length -= 1
		target = target.next[0]
	}

	skl.shrink()
}

func (skl *SkipList) Len() int { return skl.length }

func (skl *SkipList) shrink() {
	for level := skl.height - 1; level >= 0; level-- {
		if skl.head.next[level] == skl.tail {
			skl.height--
		}
	}
}
