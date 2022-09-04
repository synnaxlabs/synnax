package confluence

import (
	"github.com/arya-analytics/x/signal"
)

type TransientProvider struct {
	inlet Inlet[error]
}

func (t *TransientProvider) Transient() chan<- error { return t.inlet.Inlet() }

func (t *TransientProvider) bindTransient(err Inlet[error]) { t.inlet = err }

type bindableTransient interface {
	bindTransient(Inlet[error])
}

type bindableTransientSegment[I, O Value] interface {
	bindableTransient
	Segment[I, O]
}

type bindableTransientSource[V Value] interface {
	bindableTransient
	Source[V]
}

type bindableTransientSink[V Value] interface {
	bindableTransient
	Sink[V]
}

type transient[I, O Value] struct {
	trans Inlet[error]
	Segment[I, O]
}

type transientSource[V Value] struct {
	trans Inlet[error]
	Source[V]
}

type transientSink[V Value] struct {
	trans Inlet[error]
	Sink[V]
}

func (t transient[I, O]) Flow(ctx signal.Context, opts ...Option) {
	t.Segment.Flow(ctx, append(opts, WithClosables(t.trans))...)
}

func (t transientSource[V]) Flow(ctx signal.Context, opts ...Option) {
	t.Source.Flow(ctx, append(opts, WithClosables(t.trans))...)
}

func (t transientSink[V]) Flow(ctx signal.Context, opts ...Option) {
	t.Sink.Flow(ctx, append(opts, WithClosables(t.trans))...)
}

func InjectTransient[I, O Value](trans Inlet[error], seg bindableTransientSegment[I, O]) Segment[I, O] {
	seg.bindTransient(trans)
	return transient[I, O]{Segment: seg, trans: trans}
}

func InjectTransientSource[V Value](trans Inlet[error], source bindableTransientSource[V]) Source[V] {
	source.bindTransient(trans)
	return transientSource[V]{trans: trans, Source: source}
}

func InjectTransientSink[V Value](trans Inlet[error], sink bindableTransientSink[V]) Sink[V] {
	sink.bindTransient(trans)
	return transientSink[V]{Sink: sink, trans: trans}
}
