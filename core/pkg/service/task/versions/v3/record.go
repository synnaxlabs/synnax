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
	"encoding/json/v2"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

// NewRecordMigration returns a migration identified by key that lifts every stored
// config record of type I to type O. A record that lift rejects is stored with its task
// row under QuarantineKVPrefix, and the record and task are removed from service. The
// migration logs a warning for each one and does not fail.
func NewRecordMigration[
	I gorp.Entry[uuid.UUID],
	O gorp.Entry[uuid.UUID],
](
	key string,
	lift func(context.Context, I) (O, error),
	otg *ontology.Ontology,
) gorp.Migration {
	return migrate.New(
		key,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			records, err := collectEntries(
				ctx, gorp.WrapReader[uuid.UUID, I](tx), func(I) bool { return true },
			)
			if err != nil {
				return err
			}
			writer := gorp.WrapWriter[uuid.UUID, O](tx)
			for _, record := range records {
				lifted, liftErr := lift(ctx, record)
				if liftErr == nil {
					if err := writer.Set(ctx, lifted); err != nil {
						return err
					}
					continue
				}
				if err := quarantineRecord(
					ctx, tx, ins, otg.NewWriter(tx), record, liftErr,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// quarantineRecord moves a config record and the task that owns it into quarantine. A
// record with no task row quarantines under a row with only its key.
func quarantineRecord[I gorp.Entry[uuid.UUID]](
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	otgW ontology.Writer,
	record I,
	liftErr error,
) error {
	key := record.GorpKey()
	t := v2.Task{Key: key}
	if err := gorp.NewRetrieve[v2.Key, v2.Task]().
		Where(gorp.MatchKeys[v2.Key, v2.Task](key)).
		Entry(&t).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	config, err := encodeRecord(record)
	if err != nil {
		ins.L.Warn(
			"task config could not be encoded; staged without config",
			zap.String("key", key.String()),
			zap.Error(err),
		)
	}
	t.Config = config
	if err := quarantine(
		ctx, tx, ins, otgW, t, unconvertibleReason(liftErr),
	); err != nil {
		return err
	}
	return gorp.WrapWriter[uuid.UUID, I](tx).Delete(ctx, key)
}

func encodeRecord[I any](record I) (msgpack.EncodedJSON, error) {
	b, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	var config msgpack.EncodedJSON
	return config, json.Unmarshal(b, &config)
}
