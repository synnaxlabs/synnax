package plumber

import (
	"github.com/arya-analytics/x/address"
	cfs "github.com/arya-analytics/x/confluence"
)

// Stitch is the method a Router  uses to stitch together the segments specified in its route.
type Stitch byte

const (
	// StitchUnary is the default stitching method. It means the router will create a single stream and connected
	// it to all input (from) segments and output (to) segments.
	StitchUnary Stitch = iota
	// StitchWeave is a stitching method that means the router will create a stream for each unique combination of
	// input (from) and output (to) segments.
	StitchWeave
	// StitchConvergent is a stitching where a router creates a stream for each output (to) segment and connects it
	// to all input (from) segments.
	StitchConvergent
)

type Router[V cfs.Value] interface {
	Route(p *Pipeline) error
	PreRoute(p *Pipeline) func() error
	capacity() int
}

type UnaryRouter[V cfs.Value] struct {
	SourceTarget address.Address
	SinkTarget   address.Address
	Capacity     int
}

func (u UnaryRouter[V]) Route(p *Pipeline) error {
	return route(p, u.SourceTarget, u.SinkTarget, cfs.NewStream[V](u.Capacity))
}

func (u UnaryRouter[V]) PreRoute(p *Pipeline) func() error {
	return func() error { return u.Route(p) }
}

type MultiRouter[V cfs.Value] struct {
	SourceTargets []address.Address
	SinkTargets   []address.Address
	Capacity      int
	Stitch        Stitch
}

func (m MultiRouter[V]) Route(p *Pipeline) error {
	switch m.Stitch {
	case StitchUnary:
		return m.linear(p)
	case StitchWeave:
		return m.weave(p)
	case StitchConvergent:
		return m.convergent(p)
	}
	panic("[confluence.Router] - invalid stitch provided")
}

func (m MultiRouter[V]) PreRoute(p *Pipeline) func() error {
	return func() error { return m.Route(p) }
}

func (m MultiRouter[V]) linear(p *Pipeline) error {
	stream := cfs.NewStream[V](m.Capacity)
	stream.Acquire(int32(len(m.SourceTargets)))
	return m.iterAddresses(func(from address.Address, to address.Address) error {
		return route(p, from, to, stream)
	})
}

func (m MultiRouter[V]) weave(p *Pipeline) error {
	return m.iterAddresses(func(from, to address.Address) error {
		return UnaryRouter[V]{from, to, m.Capacity}.Route(p)
	})
}

func (m MultiRouter[V]) convergent(p *Pipeline) error {
	return iter(m.SinkTargets, func(to address.Address) error {
		stream := cfs.NewStream[V](m.Capacity)
		stream.Acquire(int32(len(m.SourceTargets)))
		return iter(m.SourceTargets, func(from address.Address) error {
			return route(p, from, to, stream)
		})
	})
}

func (m MultiRouter[V]) iterAddresses(f func(source, sink address.Address) error) error {
	return iter(m.SourceTargets, func(source address.Address) error {
		return iter(m.SinkTargets, func(sink address.Address) error {
			return f(source, sink)
		})
	})
}

func iter(targets []address.Address, f func(to address.Address) error) error {
	for _, toAddr := range targets {
		if err := f(toAddr); err != nil {
			return err
		}
	}
	return nil
}

func route[V cfs.Value](p *Pipeline, sourceTarget, sinkTarget address.Address, stream cfs.Stream[V]) error {
	source, err := GetSource[V](p, sourceTarget)
	if err != nil {
		return err
	}
	sink, err := GetSink[V](p, sinkTarget)
	if err != nil {
		return err
	}
	stream.SetInletAddress(sinkTarget)
	source.OutTo(stream)
	stream.SetOutletAddress(sourceTarget)
	sink.InFrom(stream)
	return nil
}
