// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/alias/types/v0"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns the ontology ID for an alias.
func OntologyID(r ranger.Key, ch channel.Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeRangeAlias, Key: v0.GorpKey(r, ch)}
}

// OntologyIDs returns ontology IDs for multiple aliases.
func OntologyIDs(r ranger.Key, chs []channel.Key) []ontology.ID {
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
