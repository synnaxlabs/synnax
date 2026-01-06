// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package alias implements a service for managing channel aliases on ranges.
package alias

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/zyn"
)

const keySeparator = "---"

func gorpKey(r uuid.UUID, ch channel.Key) string {
	return fmt.Sprintf("%s%s%s", r, keySeparator, ch)
}

func parseGorpKey(key string) (uuid.UUID, channel.Key, error) {
	split := strings.Split(key, keySeparator)
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

var _ gorp.Entry[string] = Alias{}

// GorpKey implements gorp.Entry.
func (a Alias) GorpKey() string { return gorpKey(a.Range, a.Channel) }

// SetOptions implements gorp.Entry.
func (a Alias) SetOptions() []any { return nil }

// OntologyType is the type used to identify aliases in the ontology.
const OntologyType ontology.Type = "range-alias"

// OntologyID returns the ontology ID for an alias.
func OntologyID(r uuid.UUID, ch channel.Key) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: gorpKey(r, ch)}
}

// OntologyIDs returns ontology IDs for multiple aliases.
func OntologyIDs(r uuid.UUID, chs []channel.Key) []ontology.ID {
	return lo.Map(chs, func(ch channel.Key, _ int) ontology.ID {
		return OntologyID(r, ch)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"range":   zyn.UUID(),
	"channel": zyn.Uint32().Coerce(),
	"alias":   zyn.String(),
})

func newResource(a Alias) ontology.Resource {
	return ontology.NewResource(
		schema,
		OntologyID(a.Range, a.Channel),
		a.Alias,
		a,
	)
}
