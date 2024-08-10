// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/synnax/pkg/label"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"regexp"
)

// Range (short for time range) is an interesting, user defined regions of time in a
// Synnax cluster. They act as a method for labeling and categorizing data.
type Range struct {
	// tx is the transaction used to execute KV operations specific to this range
	tx  gorp.Tx
	otg *ontology.Ontology
	// Key is a unique identifier for the Range. If not provided on creation, a new one
	// will be generated.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is a human-readable name for the range. This name does not need to be unique.
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
func (r Range) SetOptions() []interface{} { return nil }

func (r Range) UseTx(tx gorp.Tx) Range { r.tx = tx; return r }

func (r Range) setOntology(otg *ontology.Ontology) Range { r.otg = otg; return r }

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

func (r Range) GetMany(ctx context.Context, keys []string) ([]KVPair, error) {
	res := make([]KVPair, 0, len(keys))
	tKeys := lo.Map(keys, func(k string, _ int) string { return KVPair{Range: r.Key, Key: k}.GorpKey() })
	err := gorp.NewRetrieve[string, KVPair]().
		WhereKeys(tKeys...).
		Entries(&res).
		Exec(ctx, r.tx)
	return res, err
}

func (r Range) ListMetaData() (res []KVPair, err error) {
	err = gorp.NewRetrieve[string, KVPair]().
		Where(func(kv *KVPair) bool { return kv.Range == r.Key }).
		Entries(&res).
		Exec(context.Background(), r.tx)
	return res, nil
}

func (r Range) Set(ctx context.Context, key, value string) error {
	return gorp.NewCreate[string, KVPair]().
		Entry(&KVPair{Range: r.Key, Key: key, Value: value}).
		Exec(ctx, r.tx)
}

func (r Range) SetMany(ctx context.Context, pairs []KVPair) error {
	return gorp.NewCreate[string, KVPair]().
		Entries(&pairs).
		Exec(ctx, r.tx)
}

func (r Range) Delete(ctx context.Context, key string) error {
	return gorp.NewDelete[string, KVPair]().
		WhereKeys(KVPair{Range: r.Key, Key: key}.GorpKey()).
		Exec(ctx, r.tx)
}

func (r Range) SetAlias(ctx context.Context, ch channel.Key, al string) error {
	exists, err := gorp.NewRetrieve[channel.Key, channel.Channel]().WhereKeys(ch).Exists(ctx, r.tx)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(query.NotFound, "[range] - cannot alias non-existent channel %s", ch)
	}
	if err := gorp.NewCreate[string, alias]().
		Entry(&alias{Range: r.Key, Channel: ch, Alias: al}).
		Exec(ctx, r.tx); err != nil {
		return err
	}
	return r.otg.NewWriter(r.tx).DefineResource(ctx, AliasOntologyID(r.Key, ch))
}

func (r Range) GetAlias(ctx context.Context, ch channel.Key) (string, error) {
	var res alias
	err := gorp.NewRetrieve[string, alias]().
		WhereKeys(alias{Range: r.Key, Channel: ch}.GorpKey()).
		Entry(&res).
		Exec(ctx, r.tx)
	return res.Alias, err
}

func (r Range) ResolveAlias(ctx context.Context, al string) (channel.Key, error) {
	var res alias
	matcher := func(a *alias) bool { return a.Range == r.Key && a.Alias == al }
	rxp, err := regexp.Compile(al)
	if err == nil {
		matcher = func(a *alias) bool { return a.Range == r.Key && rxp.MatchString(a.Alias) }
	}
	err = gorp.NewRetrieve[string, alias]().
		Where(matcher).
		Entry(&res).
		Exec(ctx, r.tx)
	return res.Channel, err
}

func (r Range) SearchAliases(ctx context.Context, term string) ([]channel.Key, error) {
	ids, err := r.otg.SearchIDs(ctx, search.Request{Term: term, Type: aliasOntologyType})
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

func (r Range) DeleteAlias(ctx context.Context, ch channel.Key) error {
	return gorp.NewDelete[string, alias]().
		WhereKeys(alias{Range: r.Key, Channel: ch}.GorpKey()).
		Exec(ctx, r.tx)
}

func (r Range) ListAliases(ctx context.Context) (map[channel.Key]string, error) {
	res := make([]alias, 0)
	if err := gorp.NewRetrieve[string, alias]().
		Where(func(a *alias) bool { return a.Range == r.Key }).
		Entries(&res).
		Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	aliases := make(map[channel.Key]string, len(res))
	for _, a := range res {
		aliases[a.Channel] = a.Alias
	}
	return aliases, nil
}

func (r Range) SetLabels(ctx context.Context, labels ...uuid.UUID) error {
	w := r.otg.NewWriter(r.tx)
	for _, l := range labels {
		if err := w.DefineRelationship(
			ctx,
			OntologyID(r.Key),
			label.LabeledBy,
			label.OntologyID(l),
		); err != nil {
			return err
		}
	}
	return nil
}

func (r Range) DeleteLabels(ctx context.Context, labels ...uuid.UUID) error {
	w := r.otg.NewWriter(r.tx)
	for _, l := range labels {
		if err := w.DeleteRelationship(ctx, OntologyID(r.Key), label.LabeledBy, label.OntologyID(l)); err != nil {
			return err
		}
	}
	return nil
}

func (r Range) OntologyID() ontology.ID { return OntologyID(r.Key) }
