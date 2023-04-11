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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/query"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is used to retrieve information about Channel(s) in delta's distribution
// layer.
type Retrieve struct {
	tx   gorp.Tx
	gorp gorp.Retrieve[Key, Channel]
	keys Keys
}

func newRetrieve(tx gorp.Tx) Retrieve {
	return Retrieve{gorp: gorp.NewRetrieve[Key, Channel](), tx: tx}
}

// Entry binds the Channel that Retrieve will fill results into. This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entry(ch *Channel) Retrieve { r.gorp.Entry(ch); return r }

// Entries binds a slice that Retrieve will fill results into.  This is an identical
// interface to gorp.Retrieve.
func (r Retrieve) Entries(ch *[]Channel) Retrieve { r.gorp.Entries(ch); return r }

// WhereNodeKey filters for channels whose NodeKey attribute matches the provided
// leaseholder node ID.
func (r Retrieve) WhereNodeKey(nodeKey core.NodeKey) Retrieve {
	r.gorp.Where(func(ch *Channel) bool { return ch.NodeKey == nodeKey })
	return r
}

// WhereNames filters for channels whose Name attribute matches the provided name.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp.Where(func(ch *Channel) bool { return lo.Contains(names, ch.Name) })
	return r
}

// WhereKeys filters for channels with the provided Key. This is an identical interface
// to gorp.Retrieve.
func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.keys = append(r.keys, keys...)
	r.gorp.WhereKeys(keys...)
	return r
}

// Exec executes the query, binding
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.maybeEnrichError(r.gorp.Exec(ctx, gorp.OverrideTx(r.tx, tx)))
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
