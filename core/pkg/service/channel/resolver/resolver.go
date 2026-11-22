// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package resolver resolves the channels of a channel service into Arc symbols, and
// runs compiles that read channels against a consistent view of them.
package resolver

import (
	"context"
	"strconv"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// New returns a resolver that maps the channels of channels to Arc symbols by name or
// numeric key, for analyzing and compiling Arc expressions such as calculations. The
// resolver looks up channels in snap and sees only the channels that every filter
// matches.
func New(
	channels *channel.Service,
	snap gorp.Snapshot,
	filters ...channel.Filter,
) arc.SymbolResolver {
	return &symbolResolver{channels: channels, snap: snap, filters: filters}
}

// AllowDashedNames reports whether Arc may treat '-' as an identifier character, which
// a channel name can hold when channels does not validate names.
func AllowDashedNames(channels *channel.Service) bool {
	return !channels.ShouldValidateNames()
}

// symbolResolver resolves cluster channels into Arc symbols by name or numeric key. It
// is the dynamic resolver attached to a program root's GlobalResolver: cluster channels
// can appear or disappear at runtime, so they cannot be snapshotted into the ambient
// prelude at analysis time.
type symbolResolver struct {
	channels *channel.Service
	snap     gorp.Snapshot
	filters  []channel.Filter
}

var _ symbol.Resolver = (*symbolResolver)(nil)

func (r *symbolResolver) toSymbol(ch channel.Channel) *symbol.Symbol {
	return &symbol.Symbol{
		Name:       ch.Name,
		Kind:       symbol.KindChannel,
		Type:       types.Chan(types.FromTelem(ch.DataType)),
		ID:         int(ch.Key()),
		Renameable: !r.channels.IsBuiltIn(ch.Key()),
	}
}

func (r *symbolResolver) retrieve() channel.Retrieve {
	q := r.channels.NewRetrieve()
	for _, f := range r.filters {
		q = q.Where(f)
	}
	return q
}

// Resolve resolves a single cluster channel by name or numeric key.
func (r *symbolResolver) Resolve(
	ctx context.Context,
	name string,
) (*symbol.Symbol, error) {
	key, err := strconv.ParseUint(name, 10, 32)
	var ch channel.Channel
	q := r.retrieve().Entry(&ch)
	if err == nil {
		q = q.Where(channel.MatchKeys(channel.Key(key)))
	} else {
		q = q.Where(channel.MatchNames(name))
	}
	if err = q.Exec(ctx, r.snap); err != nil {
		return nil, err
	}
	return r.toSymbol(ch), nil
}

// Search fuzzy-searches non-internal cluster channels by name.
func (r *symbolResolver) Search(
	ctx context.Context,
	name string,
) ([]*symbol.Symbol, error) {
	var results []channel.Channel
	if err := r.retrieve().
		Where(channel.MatchInternal(false)).
		Search(name).
		Entries(&results).Exec(ctx, r.snap); err != nil {
		return nil, err
	}
	return lo.Map(results, func(item channel.Channel, _ int) *symbol.Symbol {
		return r.toSymbol(item)
	}), nil
}

// NewConsistent returns a resolver that answers each name the way r first answered it.
// Arc resolves a name several times in one analysis or compile, and a rename that
// commits part way through would otherwise give that pass both answers. Use one per
// analysis or compile. It is safe for concurrent use.
func NewConsistent(r arc.SymbolResolver) arc.SymbolResolver {
	return &consistentResolver{
		SymbolResolver: r,
		resolved:       make(map[string]resolution),
	}
}

// NewChecked returns a consistent resolver that looks up the channels of channels in
// snap and records each name it looks up in lookups, for CompileConsistent.
func NewChecked(
	channels *channel.Service,
	snap gorp.Snapshot,
	lookups *Lookups,
) arc.SymbolResolver {
	return &consistentResolver{
		SymbolResolver: New(channels, snap),
		resolved:       make(map[string]resolution),
		record: func(name string, first resolution) {
			lookups.Record(func(ctx context.Context, snap gorp.Snapshot) (bool, error) {
				return first.same(New(channels, snap).Resolve(ctx, name))
			})
		},
	}
}

type resolution struct {
	sym *symbol.Symbol
	err error
}

// same reports whether a second lookup, which returned sym and err, gives the answer of
// res: the same symbol ID, type, and name, or not found both times. It returns err when
// the second lookup fails for another reason.
func (res resolution) same(sym *symbol.Symbol, err error) (bool, error) {
	notFound := errors.Is(err, query.ErrNotFound)
	if err != nil && !notFound {
		return false, err
	}
	if res.err != nil || notFound {
		return notFound && errors.Is(res.err, query.ErrNotFound), nil
	}
	return res.sym.ID == sym.ID &&
		res.sym.Name == sym.Name &&
		types.Equal(res.sym.Type, sym.Type), nil
}

type consistentResolver struct {
	arc.SymbolResolver
	// record is called once for each name that the resolver looks up. It is nil when
	// the resolver records nothing.
	record   func(name string, first resolution)
	mu       sync.Mutex
	resolved map[string]resolution
}

func (r *consistentResolver) Resolve(
	ctx context.Context,
	name string,
) (*symbol.Symbol, error) {
	r.mu.Lock()
	res, ok := r.resolved[name]
	if !ok {
		res.sym, res.err = r.SymbolResolver.Resolve(ctx, name)
		r.resolved[name] = res
		if r.record != nil {
			r.record(name, res)
		}
	}
	r.mu.Unlock()
	if res.err != nil {
		return nil, res.err
	}
	sym := *res.sym
	return &sym, nil
}
