// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// symbolResolver resolves cluster channels into Arc symbols by name or numeric key. It
// is the dynamic resolver attached to a program root's GlobalResolver: cluster channels
// can appear or disappear at runtime, so they cannot be snapshotted into the ambient
// prelude at analysis time.
type symbolResolver struct {
	svc  *Service
	snap gorp.Snapshot
	// access returns whether the subject of the resolver can retrieve a record, or is
	// nil when the resolver sees every channel and folder.
	access func(context.Context) (func(resource.ID) bool, error)
}

var _ symbol.Resolver = (*symbolResolver)(nil)

func toSymbol(ch Channel) *symbol.Symbol {
	return &symbol.Symbol{
		Name:       ch.Name,
		Kind:       symbol.KindChannel,
		Type:       types.Chan(types.FromTelem(ch.DataType)),
		ID:         int(ch.Key()),
		Renameable: !ch.Internal,
	}
}

// retrieve returns a query that sees only what the subject of r can retrieve.
func (r *symbolResolver) retrieve(ctx context.Context) (Retrieve, error) {
	q := r.svc.NewRetrieve()
	if r.access == nil {
		return q, nil
	}
	allowed, err := r.access(ctx)
	if err != nil {
		return q, err
	}
	return q.AllowFolders(allowed).Where(Match(func(
		_ gorp.Context,
		_ Retrieve,
		ch *Channel,
	) (bool, error) {
		return allowed(ch.ResourceID()), nil
	})), nil
}

// Resolve resolves a single cluster channel by numeric key, by path when name holds
// validate.PathSeparator, or else by name in NFC. A resolved path is the name of its
// symbol. A name that several channels hold returns an AmbiguousNameError, described
// for the subject of r. Every holder of a bare name counts, and the one holder binds
// only when the subject can retrieve it.
func (r *symbolResolver) Resolve(
	ctx context.Context,
	name string,
) (*symbol.Symbol, error) {
	q, err := r.retrieve(ctx)
	if err != nil {
		return nil, err
	}
	if key, err := strconv.ParseUint(name, 10, 32); err == nil {
		var ch Channel
		if err = q.Where(MatchKeys(Key(key))).Entry(&ch).Exec(ctx, r.snap); err != nil {
			return nil, err
		}
		return toSymbol(ch), nil
	}
	path := strings.IndexByte(name, validate.PathSeparator) >= 0
	match := MatchPaths(name)
	if !path {
		// Every holder of a bare name counts, as a compile with no subject counts them.
		// Access decides only whether one holder binds.
		q, match = r.svc.NewRetrieve(), MatchNames(validate.NormalizeName(name))
	}
	var chs []Channel
	if err = q.Where(match).Entries(&chs).Exec(ctx, r.snap); err != nil {
		return nil, err
	}
	if len(chs) == 1 && !path {
		if chs, err = r.visible(ctx, chs); err != nil {
			return nil, err
		}
	}
	switch len(chs) {
	case 0:
		return nil, errors.Wrapf(query.ErrNotFound, "channel '%s'", name)
	case 1:
		sym := toSymbol(chs[0])
		if path {
			sym.Name = name
		}
		return sym, nil
	default:
		ambiguous := AmbiguousName(name, KeysFromChannels(chs))
		if r.access == nil {
			return nil, ambiguous
		}
		return nil, r.svc.DescribeAmbiguity(ctx, r.snap, ambiguous, r.access)
	}
}

// visible returns the channels of chs that the subject of r can retrieve.
func (r *symbolResolver) visible(
	ctx context.Context,
	chs []Channel,
) ([]Channel, error) {
	if r.access == nil {
		return chs, nil
	}
	allowed, err := r.access(ctx)
	if err != nil {
		return nil, err
	}
	return lo.Filter(chs, func(ch Channel, _ int) bool {
		return allowed(ch.ResourceID())
	}), nil
}

// Search fuzzy-searches non-internal cluster channels by name.
func (r *symbolResolver) Search(
	ctx context.Context,
	name string,
) ([]*symbol.Symbol, error) {
	q, err := r.retrieve(ctx)
	if err != nil {
		return nil, err
	}
	var results []Channel
	if err = q.
		Where(MatchInternal(false)).
		Search(name).
		Entries(&results).Exec(ctx, r.snap); err != nil {
		return nil, err
	}
	return lo.Map(results, func(item Channel, _ int) *symbol.Symbol {
		return toSymbol(item)
	}), nil
}

// NewConsistentResolver returns a resolver that answers each name the way r first
// answered it. Arc resolves a name several times in one analysis or compile, and a
// rename that commits part way through would otherwise give that pass both answers.
// Use one per analysis or compile. It is safe for concurrent use.
func NewConsistentResolver(r arc.SymbolResolver) arc.SymbolResolver {
	return &consistentResolver{
		SymbolResolver: r,
		resolved:       make(map[string]resolution),
	}
}

// NewCheckedResolver returns a consistent resolver that looks up channels in snap and
// records each name it looks up in lookups, for CompileConsistent.
func (s *Service) NewCheckedResolver(
	snap gorp.Snapshot,
	lookups *Lookups,
) arc.SymbolResolver {
	return &consistentResolver{
		SymbolResolver: s.NewArcSymbolResolver(snap),
		resolved:       make(map[string]resolution),
		record: func(name string, first resolution) {
			lookups.Record(func(ctx context.Context, snap gorp.Snapshot) (bool, error) {
				return first.same(s.NewArcSymbolResolver(snap).Resolve(ctx, name))
			})
		},
	}
}

type resolution struct {
	sym *symbol.Symbol
	err error
}

// same reports whether a second lookup, which returned sym and err, gives the answer of
// res: the same symbol ID, type, and name, not found both times, or the same holders of
// an ambiguous name. It returns err when the second lookup fails for another reason.
func (res resolution) same(sym *symbol.Symbol, err error) (bool, error) {
	// A name that several channels hold is an answer, the same only for the same
	// holders.
	ambiguous, isAmbiguous := errors.AsType[AmbiguousNameError](err)
	first, wasAmbiguous := errors.AsType[AmbiguousNameError](res.err)
	if isAmbiguous || wasAmbiguous {
		return isAmbiguous && wasAmbiguous &&
			set.New(ambiguous.Holders...).Equal(set.New(first.Holders...)), nil
	}
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
