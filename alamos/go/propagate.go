package alamos

import "context"

// Carrier is a propagation medium for traces, whether it be a protocol header, persisted
// storage, or something else. Carrier is passed to the Propagate and Depropagate functions.
type Carrier interface {
	// Set sets the given key to the given value.
	Set(key, value string)
	// Get gets the value for the given key.
	Get(key string) string
	// Keys lists the keys stored in this Carrier.
	Keys() []string
}

func (t *Tracer) propagate(ctx context.Context, carrier Carrier) {
	if t == nil {
		return
	}
	t.config.Propagator.Inject(ctx, carrier)
}

func (t *Tracer) depropagate(ctx context.Context, carrier Carrier) context.Context {
	if t == nil {
		return ctx
	}
	return t.config.Propagator.Extract(ctx, carrier)
}

// Propagate propagates the trace from the given context to the given Carrier.
func Propagate(ctx context.Context, carrier Carrier) {
	extract(ctx).T.propagate(ctx, carrier)
}

// Depropagate de-propagates the trace from the given Carrier to a new context
// that is the child of the given context.
func Depropagate(ctx context.Context, carrier Carrier) context.Context {
	return extract(ctx).T.depropagate(ctx, carrier)
}
