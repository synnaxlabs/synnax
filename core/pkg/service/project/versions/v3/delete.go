// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"go.uber.org/zap"
)

// DeleteConfig configures a migration that deletes stored visualizations of type Type.
type DeleteConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Name is the migration's key.
	Name string
	// Type is the resource type of the visualizations.
	Type resource.Type
	// NameOf returns the name of e.
	NameOf func(e E) string
	// Reason returns why the migration deletes e, or an empty string to keep e.
	Reason func(e E) string
}

// NewDeleteMigration returns a migration that deletes each stored visualization with a
// non-empty cfg.Reason, and logs a warning with the reason for each.
func NewDeleteMigration[K gorp.Key, E gorp.Entry[K]](
	cfg DeleteConfig[K, E],
) gorp.Migration {
	return migrate.New(
		cfg.Name,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var entries []E
			if err := gorp.NewRetrieve[K, E]().
				Entries(&entries).
				Exec(ctx, tx); err != nil {
				return err
			}
			var deleted []K
			for _, e := range entries {
				reason := cfg.Reason(e)
				if reason == "" {
					continue
				}
				ins.L.Warn(
					"deleted a visualization",
					zap.String("type", string(cfg.Type)),
					zap.String("key", gorp.FormatKey(e.GorpKey())),
					zap.String("name", cfg.NameOf(e)),
					zap.String("reason", reason),
				)
				deleted = append(deleted, e.GorpKey())
			}
			return gorp.WrapWriter[K, E](tx).Delete(ctx, deleted...)
		},
	)
}
