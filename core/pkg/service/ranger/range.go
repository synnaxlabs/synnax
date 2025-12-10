// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ranger implements a service for managing ranges in a Synnax cluster. A range
// is a user defined region of time in a Synnax cluster. They act as a method for
// labeling and categorizing data.
package ranger

import (
	"context"
	"regexp"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Range (short for time range) is an interesting, user defined region of time in a
// Synnax cluster. They act as a method for labeling and categorizing data.
type Range struct {
	tx    gorp.Tx
	otg   *ontology.Ontology
	label *label.Service
	// Key is a unique identifier for the Range. If not provided on creation, a new one
	// will be generated.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is a human-readable name for the range. This name does not need to be
	// unique.
	Name string `json:"name" msgpack:"name"`
	// TimeRange is the range of time occupied by the range.
	TimeRange telem.TimeRange `json:"time_range" msgpack:"time_range"`
	// Color is the color used to represent the range in the UI.
	Color string `json:"color" msgpack:"color"`
}

var _ gorp.Entry[uuid.UUID] = Range{}

// GorpKey implements gorp.Entry.
func (r Range) GorpKey() uuid.UUID { return r.Key }

// SetOptions implements gorp.Entry.
func (r Range) SetOptions() []any { return nil }

// UseTx binds a transaction to use for all query operations on the Range.
func (r Range) UseTx(tx gorp.Tx) Range { r.tx = tx; return r }

func (r Range) setOntology(otg *ontology.Ontology) Range { r.otg = otg; return r }

func (r Range) setLabel(label *label.Service) Range { r.label = label; return r }

// Get gets the provided key-value pair from the range.
func (r Range) Get(ctx context.Context, key string) (string, error) {
	var (
		res = KVPair{Range: r.Key, Key: key}
		err = gorp.NewRetrieve[string, KVPair]().
			WhereKeys(res.GorpKey()).
			Entry(&res).
			Exec(ctx, r.tx)
	)
	if errors.Is(err, query.NotFound) {
		return "", errors.Wrapf(err, "key %s not found on range", key)
	}
	return res.Value, err
}

// GetManyKV gets the provided key-value pairs from the range.
func (r Range) GetManyKV(ctx context.Context, keys []string) ([]KVPair, error) {
	res := make([]KVPair, 0, len(keys))
	tKeys := lo.Map(keys, func(k string, _ int) string { return KVPair{Range: r.Key, Key: k}.GorpKey() })
	err := gorp.NewRetrieve[string, KVPair]().
		WhereKeys(tKeys...).
		Entries(&res).
		Exec(ctx, r.tx)
	return res, err
}

// ListKV lists all key-value pairs on the range.
func (r Range) ListKV(ctx context.Context) (res []KVPair, err error) {
	err = gorp.NewRetrieve[string, KVPair]().
		Where(func(ctx gorp.Context, kv *KVPair) (bool, error) {
			return kv.Range == r.Key, nil
		}).
		Entries(&res).
		Exec(ctx, r.tx)
	return res, err
}

// SetKV sets the provided key-value pairs on the range.
func (r Range) SetKV(ctx context.Context, key, value string) error {
	return gorp.NewCreate[string, KVPair]().
		Entry(&KVPair{Range: r.Key, Key: key, Value: value}).
		Exec(ctx, r.tx)
}

// SetManyKV sets the provided key-value pairs on the range.
func (r Range) SetManyKV(ctx context.Context, pairs []KVPair) error {
	for i, p := range pairs {
		p.Range = r.Key
		pairs[i] = p
	}
	return gorp.NewCreate[string, KVPair]().Entries(&pairs).Exec(ctx, r.tx)
}

// DeleteKV deletes the provided key-value pair from the range. DeleteKV is idempotent,
// and will not return an error if the key does not exist.
func (r Range) DeleteKV(ctx context.Context, key string) error {
	return gorp.NewDelete[string, KVPair]().
		WhereKeys(KVPair{Range: r.Key, Key: key}.GorpKey()).
		Exec(ctx, r.tx)
}

// SetAlias sets an Alias for the provided channel on the range.
func (r Range) SetAlias(ctx context.Context, ch channel.Key, al string) error {
	exists, err := gorp.NewRetrieve[channel.Key, channel.Channel]().WhereKeys(ch).Exists(ctx, r.tx)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(query.NotFound, "[range] - cannot Alias non-existent channel %s", ch)
	}
	if err := gorp.NewCreate[string, Alias]().
		Entry(&Alias{Range: r.Key, Channel: ch, Alias: al}).
		Exec(ctx, r.tx); err != nil {
		return err
	}
	return r.otg.NewWriter(r.tx).DefineResource(ctx, AliasOntologyID(r.Key, ch))
}

// RetrieveAlias gets the Alias for the provided channel on the range.
func (r Range) RetrieveAlias(ctx context.Context, ch channel.Key) (string, error) {
	var res Alias
	err := gorp.NewRetrieve[string, Alias]().
		WhereKeys(Alias{Range: r.Key, Channel: ch}.GorpKey()).
		Entry(&res).
		Exec(ctx, r.tx)
	if errors.Is(err, query.NotFound) {
		p, pErr := r.RetrieveParent(ctx)
		if errors.Is(pErr, query.NotFound) {
			return res.Alias, err
		}
		return p.RetrieveAlias(ctx, ch)
	}
	return res.Alias, err
}

// ResolveAlias attempts to resolve the provided Alias to a channel key on the range.
func (r Range) ResolveAlias(ctx context.Context, alias string) (channel.Key, error) {
	var res Alias
	matcher := func(ctx gorp.Context, a *Alias) (bool, error) {
		return a.Range == r.Key && a.Alias == alias, nil
	}
	rxp, err := regexp.Compile(alias)
	if err == nil {
		matcher = func(ctx gorp.Context, a *Alias) (bool, error) {
			return a.Range == r.Key && rxp.MatchString(a.Alias), nil
		}
	}
	err = gorp.
		NewRetrieve[string, Alias]().
		Where(matcher).
		Entry(&res).
		Exec(ctx, r.tx)
	if errors.Is(err, query.NotFound) {
		p, pErr := r.RetrieveParent(ctx)
		if errors.Is(pErr, query.NotFound) {
			return 0, err
		}
		return p.ResolveAlias(ctx, alias)
	}
	if err != nil {
		return 0, err
	}
	return res.Channel, nil
}

// RetrieveParent returns the parent of the given range.
func (r Range) RetrieveParent(ctx context.Context) (Range, error) {
	var resources []ontology.Resource
	if err := r.otg.NewRetrieve().
		WhereIDs(r.OntologyID()).
		TraverseTo(ontology.Parents).
		WhereTypes(OntologyType).
		ExcludeFieldData(true).
		Entries(&resources).
		Exec(ctx, r.tx); err != nil {
		return Range{}, err
	}
	if len(resources) == 0 {
		return Range{}, errors.Wrapf(query.NotFound, "range %s has no parent", r.Key)
	}
	key, err := KeyFromOntologyID(resources[0].ID)
	if err != nil {
		return Range{}, err
	}
	var res Range
	if err = gorp.NewRetrieve[uuid.UUID, Range]().
		WhereKeys(key).
		Entry(&res).
		Exec(ctx, r.tx); err != nil {
		return Range{}, err
	}
	res.tx = r.tx
	res.otg = r.otg
	return res.UseTx(r.tx), nil
}

// SearchAliases searches for aliases fuzzily matching the given term.
func (r Range) SearchAliases(ctx context.Context, term string) ([]channel.Key, error) {
	ids, err := r.otg.SearchIDs(
		ctx,
		search.Request{Term: term, Type: AliasOntologyType},
	)
	if err != nil {
		return nil, err
	}
	res := make([]channel.Key, 0)
	for _, id := range ids {
		rangeKey, chKey, err := parseAliasKey(id.Key)
		if err != nil {
			return nil, err
		}
		if rangeKey == r.Key {
			res = append(res, chKey)
		}
	}
	return res, nil
}

// DeleteAlias deletes the Alias for the given channel on the range. DeleteAlias is
// idempotent, and will not return an error if the Alias does not exist.
func (r Range) DeleteAlias(ctx context.Context, ch channel.Key) error {
	return gorp.
		NewDelete[string, Alias]().
		WhereKeys(Alias{Range: r.Key, Channel: ch}.GorpKey()).
		Exec(ctx, r.tx)
}

// RetrieveAliases lists all aliases on the range.
func (r Range) RetrieveAliases(ctx context.Context) (map[channel.Key]string, error) {
	res := make(map[channel.Key]string)
	if err := r.listAliases(ctx, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (r Range) listAliases(
	ctx context.Context,
	accumulated map[channel.Key]string,
) error {
	res := make([]Alias, 0)
	if err := gorp.NewRetrieve[string, Alias]().
		Where(func(_ gorp.Context, a *Alias) (bool, error) {
			return a.Range == r.Key, nil
		}).
		Entries(&res).
		Exec(ctx, r.tx); err != nil {
		return err
	}
	for _, a := range res {
		accumulated[a.Channel] = a.Alias
	}
	p, pErr := r.RetrieveParent(ctx)
	if errors.Is(pErr, query.NotFound) {
		return nil
	} else if pErr != nil {
		return pErr
	}
	return p.listAliases(ctx, accumulated)
}

func (r Range) RetrieveLabels(ctx context.Context) ([]label.Label, error) {
	return r.label.RetrieveFor(ctx, OntologyID(r.Key), r.tx)
}

// OntologyID returns the semantic ID for this range to look it up from within the
// ontology.
func (r Range) OntologyID() ontology.ID { return OntologyID(r.Key) }
