// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	label "github.com/synnaxlabs/synnax/pkg/service/label/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	ranger "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	status "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v0"
	"github.com/synnaxlabs/x/errors"
	filter "github.com/synnaxlabs/x/filter/versions/v0"
	"go.uber.org/zap"
)

// NewMigration returns the migration that lifts stored views to v1. It turns the query
// of every range and status view into a where. It deletes every view of another type,
// and every view whose query does not decode into a valid where, along with the view's
// ontology resource and relationships.
func NewMigration(otg *ontology.Ontology) gorp.Migration {
	return migrate.New(
		"v60_typed_views",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			olds, err := collect(ctx, tx)
			if err != nil {
				return err
			}
			var (
				writer  = gorp.WrapWriter[Key, View](tx)
				deleted []Key
			)
			for _, old := range olds {
				v, err := lift(old)
				if err != nil {
					ins.L.Warn(
						"deleting view that v1 cannot hold",
						zap.Stringer("key", old.Key),
						zap.String("type", old.Type),
						zap.Error(err),
					)
					deleted = append(deleted, old.Key)
					continue
				}
				if err = writer.Set(ctx, v); err != nil {
					return err
				}
			}
			if len(deleted) == 0 {
				return nil
			}
			if err = writer.Delete(ctx, deleted...); err != nil {
				return err
			}
			ids := make([]ontology.ID, len(deleted))
			for i, k := range deleted {
				ids[i] = ontology.ID{Type: ontology.ResourceTypeView, Key: k.String()}
			}
			return otg.NewWriter(tx).DeleteResources(ctx, ids...)
		},
	)
}

type rangeQuery struct {
	HasLabels []label.Key `json:"has_labels"`
}

type statusQuery struct {
	HasLabels []label.Key      `json:"has_labels"`
	Variants  []status.Variant `json:"variants"`
}

// lift returns an error if v1 has no variant for the view's type or the view's query
// does not decode into a valid where for the variant. An empty query becomes an absent
// where.
func lift(old v0.View) (View, error) {
	base := Base{Key: old.Key, Name: old.Name}
	switch ViewType(old.Type) {
	case RangeViewType:
		var q rangeQuery
		if err := old.Query.Unmarshal(&q); err != nil {
			return View{}, err
		}
		v := RangeView{Base: base}
		if len(q.HasLabels) > 0 {
			v.Where = &ranger.Where{Labels: q.HasLabels}
		}
		return View{Variant: v}, nil
	case StatusViewType:
		var q statusQuery
		if err := old.Query.Unmarshal(&q); err != nil {
			return View{}, err
		}
		for _, v := range q.Variants {
			if !v.IsValid() {
				return View{}, errors.Newf("invalid status variant %q", v)
			}
		}
		var where status.Where
		if len(q.HasLabels) > 0 {
			where.Labels = q.HasLabels
		}
		if len(q.Variants) > 0 {
			where.Variant = &filter.Set[status.Variant]{In: q.Variants}
		}
		v := StatusView{Base: base}
		if where.Labels != nil || where.Variant != nil {
			v.Where = &where
		}
		return View{Variant: v}, nil
	}
	return View{}, errors.Newf("unsupported type %q", old.Type)
}

func collect(ctx context.Context, tx gorp.Tx) ([]v0.View, error) {
	reader := gorp.WrapReader[Key, v0.View](tx)
	views, closer, err := reader.OpenNexter(ctx)
	if err != nil {
		return nil, err
	}
	var olds []v0.View
	for v := range views {
		olds = append(olds, v)
	}
	return olds, closer.Close()
}
