// Copyright 2025 Synnax Labs, Inc.
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
	"regexp"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Retrieve is used to retrieve information about Channel(s) in the synnax distribution
// layer
type Retrieve struct {
	tx                        gorp.Tx
	gorp                      gorp.Retrieve[Key, Channel]
	otg                       *ontology.Ontology
	keys                      Keys
	searchTerm                string
	validateRetrievedChannels func(channels []Channel) ([]Channel, error)
}

// Search sets the search term for the query. Note that the fuzzy search will be executed
// before any other filters that are applied.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Entry binds the Channel that Retrieve will fill results into. This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entry(ch *Channel) Retrieve { r.gorp.Entry(ch); return r }

// Entries binds a slice that Retrieve will fill results into.  This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entries(ch *[]Channel) Retrieve { r.gorp.Entries(ch); return r }

// WhereNodeKey filters for channels whose Leaseholder attribute matches the provided
// leaseholder node Key.
func (r Retrieve) WhereNodeKey(nodeKey cluster.NodeKey) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, ch *Channel) (bool, error) {
		return ch.Leaseholder == nodeKey, nil
	})
	return r
}

// WhereIsIndex filters the query for channels that are indexes if isIndex is true, or
// are not indexes if isIndex is false.
func (r Retrieve) WhereIsIndex(isIndex bool) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, ch *Channel) (bool, error) {
		return ch.IsIndex == isIndex, nil
	}, gorp.Required())
	return r
}

// WhereVirtual filters the query for channels that are virtual if virtual is true, or are
// not virtual if virtual is false.
func (r Retrieve) WhereVirtual(virtual bool) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, ch *Channel) (bool, error) {
		isVirtual := ch.Virtual && !ch.IsCalculated()
		return isVirtual == virtual, nil
	}, gorp.Required())
	return r
}

// WhereInternal filters the query for channels that are internal if internal is true, or
// are not internal if internal is false.
func (r Retrieve) WhereInternal(internal bool) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, ch *Channel) (bool, error) {
		return ch.Internal == internal, nil
	}, gorp.Required())
	return r
}

func (r Retrieve) WhereLegacyCalculated() Retrieve {
	r.gorp.Where(func(ctx gorp.Context, ch *Channel) (bool, error) {
		return ch.IsLegacyCalculated(), nil
	}, gorp.Required())
	return r
}

// WhereDataTypes filters for channels whose DataType attribute matches the provided
// data types.
func (r Retrieve) WhereDataTypes(dataTypes ...telem.DataType) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, ch *Channel) (bool, error) {
		return lo.Contains(dataTypes, ch.DataType), nil
	})
	return r
}

// WhereNotDataTypes filters for channels whose DataType attribute does not match the
// provided data types.
func (r Retrieve) WhereNotDataTypes(dataTypes ...telem.DataType) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, ch *Channel) (bool, error) {
		return !lo.Contains(dataTypes, ch.DataType), nil
	})
	return r
}

// WhereNames filters for channels whose Key attribute matches the provided name.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	matchers := make([]func(string) bool, len(names))
	for i, name := range names {
		matchers[i] = formatNameMatcher(name)
	}
	r.gorp.Where(func(ctx gorp.Context, ch *Channel) (bool, error) {
		return lo.SomeBy(matchers, func(matcher func(string) bool) bool {
			return matcher(ch.Name)
		}), nil
	})
	return r
}

// WhereKeys filters for channels with the provided Key. This is an identical interface
// to gorp.Retrieve.
func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.keys = append(r.keys, keys...)
	r.gorp.WhereKeys(keys...)
	return r
}

// Limit limits the number of results returned by the query. This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp.Limit(limit); return r }

// Offset offsets the results returned by the query. This is an identical interface to
// gorp.Retrieve.
func (r Retrieve) Offset(offset int) Retrieve { r.gorp.Offset(offset); return r }

// Exec executes the query, binding
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	if r.searchTerm != "" {
		ids, err := r.otg.SearchIDs(ctx, search.Request{
			Type: OntologyType,
			Term: r.searchTerm,
		})
		if err != nil {
			return err
		}
		keys, err := KeysFromOntologyIDs(ids)
		if err != nil {
			return err
		}
		r = r.WhereKeys(keys...)
	}
	err := r.gorp.Exec(ctx, gorp.OverrideTx(r.tx, tx))

	entries := gorp.GetEntries[Key, Channel](r.gorp.Params).All()
	channels, vErr := r.validateRetrievedChannels(entries)
	gorp.SetEntries(r.gorp.Params, &channels)
	return errors.Combine(err, vErr)
}

// Exists checks if the query has results matching its parameters. If used in conjunction
// with WhereKeys, Exists will ONLY return true if ALL the keys have a matching Channel.
// Otherwise, Exists returns true if the query has ANY results.
func (r Retrieve) Exists(ctx context.Context, tx gorp.Tx) (bool, error) {
	return r.gorp.Exists(ctx, gorp.OverrideTx(r.tx, tx))
}

func formatNameMatcher(name string) func(name string) bool {
	if !strings.HasPrefix(name, "^") && !strings.HasSuffix(name, "$") {
		name = "^" + name + "$"
	}
	rx, err := regexp.Compile(name)
	if err != nil {
		return func(s string) bool { return s == name }
	}
	return rx.MatchString

}
