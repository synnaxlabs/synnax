// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import "github.com/synnaxlabs/gorp"

// Retrieve is a query that retrieves secrets, built with chained calls.
type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[Key, Secret]
	search *gorp.SearchIndex[Key, Secret]
}

// Search filters the secrets to those whose name matches term, closest match first. An
// empty term adds no filter.
func (r Retrieve) Search(term string) Retrieve {
	if term == "" {
		return r
	}
	r.gorp = r.gorp.Where(r.search.Filter(term))
	return r
}
