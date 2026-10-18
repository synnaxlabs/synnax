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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"go.uber.org/zap"
)

// NewMigration returns the migration that lifts stored views to v1. It types the query
// of every range and status view, and deletes every view of another type along with its
// ontology resource and relationships.
func NewMigration(otg *ontology.Ontology) migrate.Migration {
	return gorp.NewMigration(
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
				v, ok, err := lift(old)
				if err != nil {
					return errors.Wrapf(err, "view %s", old.Key)
				}
				if !ok {
					ins.L.Info(
						"deleting view of unsupported type",
						zap.Stringer("key", old.Key),
						zap.String("type", old.Type),
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

// lift returns false if v1 has no variant for the view's type.
func lift(old v0.View) (View, bool, error) {
	base := Base{Key: old.Key, Name: old.Name}
	switch ViewType(old.Type) {
	case RangeViewType:
		var q RangeQuery
		if err := old.Query.Unmarshal(&q); err != nil {
			return View{}, false, err
		}
		return View{Variant: RangeView{Base: base, Query: q}}, true, nil
	case StatusViewType:
		var q StatusQuery
		if err := old.Query.Unmarshal(&q); err != nil {
			return View{}, false, err
		}
		return View{Variant: StatusView{Base: base, Query: q}}, true, nil
	}
	return View{}, false, nil
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
