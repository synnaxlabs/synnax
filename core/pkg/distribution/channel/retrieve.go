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
	"regexp"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Retrieve is used to retrieve information about Channel(s) in the synnax distribution
// layer
type Retrieve struct {
	tx                        gorp.Tx
	gorp                      gorp.Retrieve[Key, Channel]
	search                    *search.Index
	validateRetrievedChannels func([]Channel) ([]Channel, error)
	searchTerm                string
	keys                      Keys
}

// Search sets the search term for the query. Note that the fuzzy search will be executed
// before any other filters that are applied.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Entry binds the Channel that Retrieve will fill results into. This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entry(ch *Channel) Retrieve { r.gorp = r.gorp.Entry(ch); return r }

// Entries binds a slice that Retrieve will fill results into.  This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entries(ch *[]Channel) Retrieve { r.gorp = r.gorp.Entries(ch); return r }

// Where adds the provided filters to the query, ANDing them with any existing filters.
func (r Retrieve) Where(filters ...gorp.Filter[Key, Channel]) Retrieve {
	r.gorp = r.gorp.Where(filters...)
	return r
}

// WhereNodeKey returns a filter for channels whose Leaseholder attribute matches the
// provided leaseholder node Key.
func WhereNodeKey(nodeKey cluster.NodeKey) gorp.Filter[Key, Channel] {
	return gorp.Match[Key, Channel](func(_ gorp.Context, ch *Channel) (bool, error) {
		return ch.Leaseholder == nodeKey, nil
	})
}

// WhereIsIndex returns a filter for channels that are indexes if isIndex is true, or
// are not indexes if isIndex is false.
func WhereIsIndex(isIndex bool) gorp.Filter[Key, Channel] {
	return gorp.Match[Key, Channel](func(_ gorp.Context, ch *Channel) (bool, error) {
		return ch.IsIndex == isIndex, nil
	})
}

// WhereVirtual returns a filter for channels that are virtual if virtual is true, or are
// not virtual if virtual is false.
func WhereVirtual(virtual bool) gorp.Filter[Key, Channel] {
	return gorp.Match[Key, Channel](func(_ gorp.Context, ch *Channel) (bool, error) {
		isVirtual := ch.Virtual && !ch.IsCalculated()
		return isVirtual == virtual, nil
	})
}

// WhereInternal returns a filter for channels that are internal if internal is true, or
// are not internal if internal is false.
func WhereInternal(internal bool) gorp.Filter[Key, Channel] {
	return gorp.Match[Key, Channel](func(_ gorp.Context, ch *Channel) (bool, error) {
		return ch.Internal == internal, nil
	})
}

// WhereDataTypes returns a filter for channels whose DataType attribute matches the
// provided data types.
func WhereDataTypes(dataTypes ...telem.DataType) gorp.Filter[Key, Channel] {
	return gorp.Match[Key, Channel](func(_ gorp.Context, ch *Channel) (bool, error) {
		return lo.Contains(dataTypes, ch.DataType), nil
	})
}

// WhereNotDataTypes returns a filter for channels whose DataType attribute does not
// match the provided data types.
func WhereNotDataTypes(dataTypes ...telem.DataType) gorp.Filter[Key, Channel] {
	return gorp.Match[Key, Channel](func(_ gorp.Context, ch *Channel) (bool, error) {
		return !lo.Contains(dataTypes, ch.DataType), nil
	})
}

// WhereCalculated returns a filter for channels that have a non-empty Expression field.
func WhereCalculated() gorp.Filter[Key, Channel] {
	return gorp.Match[Key, Channel](func(_ gorp.Context, ch *Channel) (bool, error) {
		return ch.IsCalculated(), nil
	})
}

// WhereNames returns a filter for channels whose Name attribute matches the provided
// names.
func WhereNames(names ...string) gorp.Filter[Key, Channel] {
	matchers := make([]func(string) bool, len(names))
	for i, name := range names {
		matchers[i] = formatNameMatcher(name)
	}
	return gorp.Match[Key, Channel](func(_ gorp.Context, ch *Channel) (bool, error) {
		return lo.SomeBy(matchers, func(matcher func(string) bool) bool {
			return matcher(ch.Name)
		}), nil
	})
}

// WhereKeys filters for channels with the provided Key. This is an identical interface
// to gorp.Retrieve.
func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.keys = append(r.keys, keys...)
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// Limit limits the number of results returned by the query. This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// Offset offsets the results returned by the query. This is an identical interface to
// gorp.Retrieve.
func (r Retrieve) Offset(offset int) Retrieve { r.gorp = r.gorp.Offset(offset); return r }

// Exec executes the query, binding
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	if r.searchTerm != "" {
		ids, err := r.search.Search(ctx, search.Request{
			Type: ontology.ResourceTypeChannel,
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
	entries := r.gorp.GetEntries()
	channels, vErr := r.validateRetrievedChannels(entries.All())
	r.gorp = r.gorp.Entries(&channels)
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
