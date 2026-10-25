// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
)

// Retrieve is used to retrieve views from the database using a builder pattern.
type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[Key, View]
	search *gorp.SearchIndex[Key, View]
}

// Search filters the views to those whose name matches term, closest match first. An
// empty term adds no filter.
func (r Retrieve) Search(term string) Retrieve {
	if term == "" {
		return r
	}
	r.gorp = r.gorp.Where(r.search.Filter(term))
	return r
}

// MatchTypes returns a filter for views whose type matches any of the provided types.
func MatchTypes(types ...ViewType) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, v *View) (bool, error) {
		return lo.Contains(types, v.Type()), nil
	})
}
