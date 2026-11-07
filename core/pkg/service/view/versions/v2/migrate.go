// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	ranger "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
	status "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	v1 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v1"
	"github.com/synnaxlabs/x/errors"
	filter "github.com/synnaxlabs/x/filter/versions/v0"
)

// Migration turns the query of every stored view into a where. An empty query becomes
// an absent where.
var Migration = migrate.NewEntry(
	"v61_view_where",
	func(_ context.Context, old v1.View) (View, error) {
		switch variant := old.Variant.(type) {
		case v1.RangeView:
			return View{Variant: RangeView{
				Base:  variant.Base,
				Where: rangeWhere(variant.Query),
			}}, nil
		case v1.StatusView:
			return View{Variant: StatusView{
				Base:  variant.Base,
				Where: statusWhere(variant.Query),
			}}, nil
		}
		return View{}, errors.Newf("unknown view variant %T", old.Variant)
	},
)

func rangeWhere(q v1.RangeQuery) *ranger.Where {
	if len(q.HasLabels) == 0 {
		return nil
	}
	return &ranger.Where{Labels: q.HasLabels}
}

// statusWhere leaves an empty list unset, since a where holding an empty list matches
// nothing.
func statusWhere(q v1.StatusQuery) *status.Where {
	var w status.Where
	if len(q.HasLabels) > 0 {
		w.Labels = q.HasLabels
	}
	if len(q.Variants) > 0 {
		w.Variant = &filter.Set[status.Variant]{In: q.Variants}
	}
	if w.Labels == nil && w.Variant == nil {
		return nil
	}
	return &w
}
