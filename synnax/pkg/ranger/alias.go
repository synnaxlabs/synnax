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
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

const aliasKeySeparator = "---"

func aliasKey(r uuid.UUID, c channel.Key) string {
	return fmt.Sprintf("%s%s%s", r, aliasKeySeparator, c)
}

func parseAliasKey(s string) (uuid.UUID, channel.Key, error) {
	split := strings.Split(s, aliasKeySeparator)
	if len(split) != 2 {
		return uuid.Nil, 0, errors.Newf("[alias] - invalid key")
	}
	r, err := uuid.Parse(split[0])
	if err != nil {
		return uuid.Nil, 0, errors.Wrapf(err, "[alias] - invalid range")
	}
	c, err := channel.ParseKey(split[1])
	if err != nil {
		return uuid.Nil, 0, errors.Wrapf(err, "[alias] - invalid channel")
	}
	return r, c, nil
}

type alias struct {
	Range   uuid.UUID   `json:"range" msgpack:"range"`
	Channel channel.Key `json:"channel" msgpack:"channel"`
	Alias   string      `json:"alias" msgpack:"alias"`
}

var _ gorp.Entry[string] = alias{}

// GorpKey implements gorp.Entry.
func (a alias) GorpKey() string { return aliasKey(a.Range, a.Channel) }

// SetOptions implements gorp.Entry.
func (a alias) SetOptions() []interface{} {
	// TODO: Figure out if we should return leaseholder here.
	return nil
}

const aliasOntologyType ontology.Type = "range-alias"

func AliasOntologyID(r uuid.UUID, c channel.Key) ontology.ID {
	return ontology.ID{Type: aliasOntologyType, Key: aliasKey(r, c)}
}

func AliasOntologyIDs(r uuid.UUID, chs []channel.Key) []ontology.ID {
	ids := make([]ontology.ID, 0, len(chs))
	for _, ch := range chs {
		ids = append(ids, AliasOntologyID(r, ch))
	}
	return ids
}

var _aliasSchema = &ontology.Schema{
	Type: aliasOntologyType,
	Fields: map[string]schema.Field{
		"range":   {Type: schema.String},
		"channel": {Type: schema.Uint32},
		"alias":   {Type: schema.String},
	},
}

func newAliasResource(a alias) schema.Resource {
	e := schema.NewResource(_aliasSchema, AliasOntologyID(a.Range, a.Channel), a.Alias)
	schema.Set(e, "range", a.Range.String())
	schema.Set(e, "channel", uint32(a.Channel))
	schema.Set(e, "alias", a.Alias)
	return e
}

type (
	aliasOntologyService struct{ db *gorp.DB }

	aliasChange = changex.Change[string, alias]
)

var _ ontology.Service = (*aliasOntologyService)(nil)

// Schema implements ontology.Service.
func (s *aliasOntologyService) Schema() *ontology.Schema { return _aliasSchema }

// RetrieveResource implements ontology.Service.
func (s *aliasOntologyService) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (schema.Resource, error) {
	rangeKey, channelKey, err := parseAliasKey(key)
	if err != nil {
		return schema.Resource{}, nil
	}
	var res alias
	err = gorp.NewRetrieve[string, alias]().
		WhereKeys(alias{Range: rangeKey, Channel: channelKey}.GorpKey()).
		Entry(&res).
		Exec(ctx, tx)
	return newAliasResource(res), err
}

func translateAliasChange(c aliasChange) schema.Change {
	return schema.Change{
		Variant: c.Variant,
		Key:     AliasOntologyID(c.Value.Range, c.Value.Channel),
		Value:   newAliasResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *aliasOntologyService) OnChange(f func(ctx context.Context, nexter iter.Nexter[schema.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[string, alias]) {
		f(ctx, iter.NexterTranslator[aliasChange, schema.Change]{
			Wrap: reader, Translate: translateAliasChange,
		})
	}
	return gorp.Observe[string, alias](s.db).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *aliasOntologyService) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[string, alias](s.db).OpenNexter()
	return iter.NexterCloserTranslator[alias, schema.Resource]{
		Wrap:      n,
		Translate: newAliasResource,
	}, err
}
