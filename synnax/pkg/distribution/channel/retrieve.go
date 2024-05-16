// Copyright 2023 Synnax Labs, Inc.
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

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Retrieve is used to retrieve information about Channel(s) in delta's distribution
// layer.
type Retrieve struct {
	tx                        gorp.Tx
	gorp                      gorp.Retrieve[Key, Channel]
	otg                       *ontology.Ontology
	keys                      Keys
	searchTerm                string
	validateRetrievedChannels func(channels []Channel) ([]Channel, error)
}

func newRetrieve(tx gorp.Tx, otg *ontology.Ontology, validateRetrievedChannels func([]Channel) ([]Channel, error)) Retrieve {
	return Retrieve{
		gorp:                      gorp.NewRetrieve[Key, Channel](),
		tx:                        tx,
		otg:                       otg,
		validateRetrievedChannels: validateRetrievedChannels,
	}
}

func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Entry binds the Channel that Retrieve will fill results into. This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entry(ch *Channel) Retrieve { r.gorp.Entry(ch); return r }

// Entries binds a slice that Retrieve will fill results into.  This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entries(ch *[]Channel) Retrieve { r.gorp.Entries(ch); return r }

// WhereNodeKey filters for channels whose Leaseholder attribute matches the provided
// leaseholder node Key.
func (r Retrieve) WhereNodeKey(nodeKey core.NodeKey) Retrieve {
	r.gorp.Where(func(ch *Channel) bool { return ch.Leaseholder == nodeKey })
	return r
}

// WhereNames filters for channels whose Key attribute matches the provided name.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	matchers := make([]func(string) bool, len(names))
	for i, name := range names {
		matchers[i] = formatNameMatcher(name)
	}
	r.gorp.Where(func(ch *Channel) bool {
		return lo.SomeBy(matchers, func(matcher func(string) bool) bool {
			return matcher(ch.Name)
		})
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
func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp.Offset(offset)
	return r
}

// Exec executes the query, binding
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) (err error) {
	if r.searchTerm != "" {
		ids, err := r.otg.SearchIDs(ctx, search.Request{
			Type: ontologyType,
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
	err = r.maybeEnrichError(r.gorp.Exec(ctx, gorp.OverrideTx(r.tx, tx)))

	entries := gorp.GetEntries[Key, Channel](r.gorp.Params).All()
	channels, vErr := r.validateRetrievedChannels(entries)
	gorp.SetEntries[Key, Channel](r.gorp.Params, &channels)
	return errors.CombineErrors(err, vErr)
}

// Exists checks if the query has results matching its parameters. If used in conjunction
// with WhereKeys, Exists will ONLY return true if ALL the keys have a matching Channel.
// Otherwise, Exists returns true if the query has ANY results.
func (r Retrieve) Exists(ctx context.Context, tx gorp.Tx) (bool, error) {
	return r.gorp.Exists(ctx, gorp.OverrideTx(r.tx, tx))
}

func (r Retrieve) maybeEnrichError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, query.NotFound) && len(r.keys) > 0 {
		channels := gorp.GetEntries[Key, Channel](r.gorp.Params).All()
		diff, _ := r.keys.Difference(KeysFromChannels(channels))
		return errors.Wrapf(query.NotFound, "channels with keys %v not found", diff)
	}
	return err
}

func formatNameMatcher(name string) func(name string) bool {
	if !strings.HasPrefix(name, "^") && !strings.HasSuffix(name, "$") {
		name = "^" + name + "$"
	}
	rx, err := regexp.Compile(name)
	if err != nil {
		return func(s string) bool { return s == name }
	} else {
		return rx.MatchString
	}
}
