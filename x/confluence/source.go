package confluence

import (
	"context"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
)

// AbstractMultiSource is a basic implementation of a Source that can send values to
// multiple Outlet(s). It implements an empty Flow method, as sources are typically
// driven by external events. The user can define a custom Flow method if they wish to
// drive the source themselves.
type AbstractMultiSource[V Value] struct {
	Out []Inlet[V]
}

// OutTo implements the Source interface.
func (ams *AbstractMultiSource[V]) OutTo(inlets ...Inlet[V]) { ams.Out = append(ams.Out, inlets...) }

// SendToEach sends the provided value to each Inlet in the Source.
func (ams *AbstractMultiSource[V]) SendToEach(ctx context.Context, v V) error {
	for _, inlet := range ams.Out {
		if err := signal.SendUnderContext(ctx, inlet.Inlet(), v); err != nil {
			return err
		}
	}
	return nil
}

// CloseInlets implements the InletCloser interface.
func (ams *AbstractMultiSource[V]) CloseInlets() {
	for _, inlet := range ams.Out {
		inlet.Close()
	}
}

// AbstractUnarySource is a basic implementation of a Source that sends values to a
// single Outlet. The user can define a custom Flow method if they wish to
// drive the source themselves.
type AbstractUnarySource[O Value] struct {
	Out Inlet[O]
}

// OutTo implements the Source interface.
func (aus *AbstractUnarySource[O]) OutTo(inlets ...Inlet[O]) {
	if len(inlets) != 1 {
		panic("[confluence.AbstractUnarySource] -  must have exactly one outlet")
	}
	aus.Out = inlets[0]
}

// CloseInlets implements the InletCloser interface.
func (aus *AbstractUnarySource[O]) CloseInlets() { aus.Out.Close() }

// AbstractAddressableSource is an implementation of a Source that stores its Inlet(s) in an
// addressable map. This is ideal for use cases where the address of an Inlet is
// relevant to the routing of the value (such as a Switch).
type AbstractAddressableSource[O Value] struct {
	// Out is an address map of all Inlet(s) reachable by the Source.
	Out map[address.Address]Inlet[O]
}

// OutTo implements the Source interface. Inlets provided must have a valid Inlet.
// InletAddress. If two inlets are provided with the same address, the last Inlet
// will override the previous one.
func (aas *AbstractAddressableSource[O]) OutTo(inlets ...Inlet[O]) {
	if aas.Out == nil {
		aas.Out = make(map[address.Address]Inlet[O])
	}
	for _, inlet := range inlets {
		aas.Out[inlet.InletAddress()] = inlet
	}
}

// Send sends a value to the target address. Returns add
func (aas *AbstractAddressableSource[O]) Send(ctx context.Context, target address.Address, v O) error {
	inlet, ok := aas.Out[target]
	if !ok {
		return address.TargetNotFound(target)
	}
	return signal.SendUnderContext(ctx, inlet.Inlet(), v)
}

// CloseInlets closes all Inlet(s) provided to AbstractAddressableSource.OutTo.
func (aas *AbstractAddressableSource[O]) CloseInlets() {
	for _, inlet := range aas.Out {
		inlet.Close()
	}
}
