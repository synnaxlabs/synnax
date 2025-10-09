// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const aliasKeySeparator = "---"

func aliasKey(r uuid.UUID, ch channel.Key) string {
	return fmt.Sprintf("%s%s%s", r, aliasKeySeparator, ch)
}

func parseAliasKey(key string) (uuid.UUID, channel.Key, error) {
	split := strings.Split(key, aliasKeySeparator)
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

type Alias struct {
	Range   uuid.UUID   `json:"range" msgpack:"range"`
	Channel channel.Key `json:"channel" msgpack:"channel"`
	Alias   string      `json:"alias" msgpack:"alias"`
}

var _ gorp.Entry[string] = Alias{}

// GorpKey implements gorp.Entry.
func (a Alias) GorpKey() string { return aliasKey(a.Range, a.Channel) }

// SetOptions implements gorp.Entry.
func (a Alias) SetOptions() []any { return nil }

const aliasOntologyType ontology.Type = "range-alias"

func AliasOntologyID(r uuid.UUID, ch channel.Key) ontology.ID {
	return ontology.ID{Type: aliasOntologyType, Key: aliasKey(r, ch)}
}

func AliasOntologyIDs(r uuid.UUID, chs []channel.Key) []ontology.ID {
	return lo.Map(chs, func(ch channel.Key, _ int) ontology.ID {
		return AliasOntologyID(r, ch)
	})
}

var aliasSchema = zyn.Object(map[string]zyn.Schema{
	"range":   zyn.UUID(),
	"channel": zyn.Uint32().Coerce(),
	"alias":   zyn.String(),
})

func newAliasResource(a Alias) ontology.Resource {
	return core.NewResource(
		aliasSchema,
		AliasOntologyID(a.Range, a.Channel),
		a.Alias,
		a,
	)
}

type (
	aliasOntologyService struct{ db *gorp.DB }

	aliasChange = xchange.Change[string, Alias]
)

var _ ontology.Service = (*aliasOntologyService)(nil)

func (a *aliasOntologyService) Type() ontology.Type { return aliasOntologyType }

// Schema implements ontology.Service.
func (s *aliasOntologyService) Schema() zyn.Schema { return aliasSchema }

// RetrieveResource implements ontology.Service.
func (s *aliasOntologyService) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	rangeKey, channelKey, err := parseAliasKey(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var res Alias
	if err = gorp.NewRetrieve[string, Alias]().
		WhereKeys(Alias{Range: rangeKey, Channel: channelKey}.GorpKey()).
		Entry(&res).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newAliasResource(res), nil
}

func translateAliasChange(c aliasChange) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     AliasOntologyID(c.Value.Range, c.Value.Channel),
		Value:   newAliasResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *aliasOntologyService) OnChange(
	f func(context.Context, iter.Nexter[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[string, Alias]) {
		f(ctx, iter.NexterTranslator[aliasChange, ontology.Change]{
			Wrap: reader, Translate: translateAliasChange,
		})
	}
	return gorp.Observe[string, Alias](s.db).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *aliasOntologyService) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[string, Alias](s.db).OpenNexter()
	if err != nil {
		return nil, err
	}
	return iter.NexterCloserTranslator[Alias, ontology.Resource]{
		Wrap:      n,
		Translate: newAliasResource,
	}, nil
}
