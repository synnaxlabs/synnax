// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"slices"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/drift"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/zyn"
)

// channelParams maps the name of each flow function to the channel params that a graph
// node sets in its inputs.
type channelParams map[string]types.Params

// newChannelParams returns the channel params of each flow function in syms, module
// members by their qualified name.
func newChannelParams(syms []*symbol.Symbol) channelParams {
	params := make(channelParams)
	var add func(name string, sym *symbol.Symbol)
	add = func(name string, sym *symbol.Symbol) {
		if sym.Kind == symbol.KindModule {
			for _, member := range sym.Children() {
				add(name+"."+member.Name, member)
			}
			return
		}
		if sym.Kind != symbol.KindFunction {
			return
		}
		if p := channelInputs(sym.Type.Inputs); len(p) > 0 {
			params[name] = p
		}
	}
	for _, sym := range syms {
		add(sym.Name, sym)
	}
	return params
}

// channelInputs returns the params in inputs that take a channel.
func channelInputs(inputs types.Params) types.Params {
	return slices.DeleteFunc(slices.Clone(inputs), func(p types.Param) bool {
		return p.Type.Kind != types.KindChan
	})
}

// nodeKeys appends to keys the channel that each node of g sets on a channel param in
// direction.
func (c channelParams) nodeKeys(
	keys []channel.Key,
	g graph.Graph,
	direction types.ChanDirection,
) []channel.Key {
	c.eachNodeKey(g, direction, func(_ msgpack.EncodedJSON, _ string, k channel.Key) {
		keys = append(keys, k)
	})
	return keys
}

// eachNodeKey calls f with the inputs, param name, and channel of each channel param in
// direction that a node of g sets. A param with no direction reads. A function of g
// shadows a flow function of the same name, and a value that is no channel key counts
// for nothing.
func (c channelParams) eachNodeKey(
	g graph.Graph,
	direction types.ChanDirection,
	f func(inputs msgpack.EncodedJSON, param string, k channel.Key),
) {
	for _, n := range g.Nodes {
		inputs := g.Inputs[n.Key]
		name, _ := inputs["type"].(string)
		params, ok := c[name]
		if i := slices.IndexFunc(g.Functions, func(f ir.Function) bool {
			return f.Key == name
		}); i >= 0 {
			params, ok = g.Functions[i].Inputs, true
		}
		if !ok {
			continue
		}
		for _, p := range params {
			v, set := inputs[p.Name]
			d := p.Type.ChanDirection
			if d == types.ChanDirectionNone {
				d = types.ChanDirectionRead
			}
			if !set || p.Type.Kind != types.KindChan || d&direction == 0 {
				continue
			}
			var k uint32
			if zyn.Uint32().Coerce().Parse(v, &k) == nil && k != 0 {
				f(inputs, p.Name, channel.Key(k))
			}
		}
	}
}

// newChannelReference returns the reference from each Arc to the channels it uses in
// direction: the keys that list returns from its Base, and the channels that its graph
// nodes set on channel params in direction. A deleted channel stays in the Arc, and a
// detach removes the target from the list and clears each node input that holds it.
func newChannelReference(
	name string,
	direction types.ChanDirection,
	tag uint8,
	list func(*Base) *[]channel.Key,
	params channelParams,
) *gorp.KeyReference[Key, Arc, channel.Key] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Arc, channel.Key]{
		Name:     name,
		Table:    string(resource.TypeChannel),
		OnDelete: gorp.Weak,
		Kind:     gorp.Use,
		Tag:      tag,
		Targets: func(a *Arc) []channel.Key {
			b := a.Base()
			src, ok := b.Source.Variant.(GraphSource)
			if !ok {
				return *list(&b)
			}
			keys := params.nodeKeys(slices.Clone(*list(&b)), src.Graph, direction)
			slices.Sort(keys)
			return slices.Compact(keys)
		},
		Detach: func(a *Arc, t channel.Key) {
			b := a.Base()
			held := list(&b)
			*held = slices.DeleteFunc(*held, func(k channel.Key) bool { return k == t })
			a.SetBase(b)
			src, ok := b.Source.Variant.(GraphSource)
			if !ok {
				return
			}
			params.eachNodeKey(src.Graph, direction, func(
				inputs msgpack.EncodedJSON,
				param string,
				k channel.Key,
			) {
				if k == t {
					delete(inputs, param)
				}
			})
		},
	})
}

// newChannelReferences returns the references from each Arc to the channels it reads
// and writes.
func newChannelReferences(params channelParams) []gorp.Reference[Key, Arc] {
	return []gorp.Reference[Key, Arc]{
		newChannelReference(
			"reads",
			types.ChanDirectionRead,
			channel.ReadTag,
			func(b *Base) *[]channel.Key { return &b.Reads },
			params,
		),
		newChannelReference(
			"writes",
			types.ChanDirectionWrite,
			channel.WriteTag,
			func(b *Base) *[]channel.Key { return &b.Writes },
			params,
		),
	}
}

// newDeployedReferences returns the references from each drift record to the channels
// that the program its task last ran reads and writes. A deleted channel stays in the
// record.
func newDeployedReferences() []gorp.Reference[drift.Key, drift.Record] {
	deployed := func(
		name string,
		tag uint8,
		list func(*drift.Record) []channel.Key,
	) gorp.Reference[drift.Key, drift.Record] {
		return gorp.NewKeyReference(
			gorp.KeyReferenceConfig[drift.Key, drift.Record, channel.Key]{
				Name:     name,
				Table:    string(resource.TypeChannel),
				OnDelete: gorp.Weak,
				Kind:     gorp.Use,
				Tag:      tag,
				Targets:  list,
			},
		)
	}
	return []gorp.Reference[drift.Key, drift.Record]{
		deployed("reads", channel.ReadTag, func(r *drift.Record) []channel.Key {
			return r.Reads
		}),
		deployed("writes", channel.WriteTag, func(r *drift.Record) []channel.Key {
			return r.Writes
		}),
	}
}
