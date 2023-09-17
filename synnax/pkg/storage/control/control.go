package control

import (
	"github.com/synnaxlabs/x/telem"
	"go/types"
	"math"
	"sync"
)

type Authority uint8

const (
	Absolute Authority = math.MaxUint8
)

type Service[T comparable] struct {
	sync.RWMutex
	gates map[*Gate[T]]types.Nil
}

func NewService[T comparable]() *Service[T] {
	return &Service[T]{gates: make(map[*Gate[T]]types.Nil)}
}

type Gate[T comparable] struct {
	closed    bool
	svc       *Service[T]
	timeRange telem.TimeRange
	entities  map[T]Authority
}

func (s *Service[T]) OpenGate(tr telem.TimeRange) *Gate[T] {
	g := &Gate[T]{svc: s, timeRange: tr, entities: make(map[T]Authority)}
	s.Lock()
	s.gates[g] = types.Nil{}
	s.Unlock()
	return g
}

const closed = "[control.Gate] - closed"

func (g *Gate[T]) Close() {
	g.svc.Lock()
	if g.closed {
		g.svc.Unlock()
		panic(closed)
	}
	if g.closed {
		panic(closed)
	}
	delete(g.svc.gates, g)
	g.closed = true
	g.svc.Unlock()
}

func (g *Gate[T]) Set(e []T, auth []Authority) {
	g.svc.Lock()
	if g.closed {
		g.svc.Unlock()
		panic(closed)
	}
	for i, key := range e {
		g.entities[key] = auth[i]
	}
	g.svc.Unlock()
}

func (g *Gate[T]) Delete(key T) {
	g.svc.Lock()
	if g.closed {
		g.svc.Unlock()
		panic(closed)
	}
	delete(g.entities, key)
	g.svc.Unlock()
}

func (g *Gate[T]) Check(entities []T) (failed []T) {
	g.svc.RLock()
	if g.closed {
		g.svc.RUnlock()
		panic("[control.Gate] - closed")
	}
	for _, key := range entities {
		auth, ok := g.entities[key]
		if !ok {
			failed = append(failed, key)
			continue
		}
		for other := range g.svc.gates {
			if other.timeRange.OverlapsWith(g.timeRange) {
				if other.entities[key] > auth {
					failed = append(failed, key)
					break
				}
			}
		}
	}
	g.svc.RUnlock()
	return
}
