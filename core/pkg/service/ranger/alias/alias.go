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
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/zyn"
)

const keySeparator = "---"

func parseGorpKey(key string) (ranger.Key, channel.Key, error) {
	split := strings.Split(key, keySeparator)
	if len(split) != 2 {
		return uuid.Nil(), 0, errors.Newf("[alias] - invalid key")
	}
	r, err := uuid.Parse(split[0])
	if err != nil {
		return uuid.Nil(), 0, errors.Wrapf(err, "[alias] - invalid range")
	}
	c, err := channel.ParseKey(split[1])
	if err != nil {
		return uuid.Nil(), 0, errors.Wrapf(err, "[alias] - invalid channel")
	}
	return r, c, nil
}

// ResourceID returns the resource ID for an alias.
func ResourceID(r ranger.Key, ch channel.Key) resource.ID {
	return resource.ID{
		Type: resource.TypeRangeAlias,
		Key:  Alias{Range: r, Channel: ch}.GorpKey(),
	}
}

// ResourceIDs returns resource IDs for multiple aliases.
func ResourceIDs(r ranger.Key, chs []channel.Key) []resource.ID {
	return lo.Map(chs, func(ch channel.Key, _ int) resource.ID {
		return ResourceID(r, ch)
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
		ResourceID(a.Range, a.Channel),
		a.Alias,
		a,
	)
}

func newSearchIndex() *gorp.SearchIndex[string, Alias] {
	return gorp.NewSearchIndex("search", func(a *Alias) []string {
		return []string{a.Alias}
	})
}
