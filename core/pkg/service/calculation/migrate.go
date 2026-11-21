// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv2 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/telem"
	xuuid "github.com/synnaxlabs/x/uuid"
)

// migrationNamespace is the UUIDv5 namespace of the key that Migration gives the
// calculation of a channel.
var migrationNamespace = uuid.MustParse("3d7c1f2a-9e44-4c1b-8a55-2f6b0e9d4c71")

// migratedKey returns the key that Migration gives the calculation of the channel with
// key.
func migratedKey(key channel.Key) Key {
	return xuuid.NewV5(migrationNamespace, key.String())
}

// Migration returns the migration that builds the calculation record of each channel
// that the channel migrations staged, keyed by a UUIDv5 of the channel key, and deletes
// the staged entry and the status of each one. A reset channel that issuedAtOpen says
// the Core never issued
// becomes absent. An operation of no type is dropped, with a warning status that names
// the channel. Migration reads the channel and status tables, so it runs after both
// open.
func (s *Service) Migration(
	statuses *status.Service,
	issuedAtOpen func(channel.Key) bool,
) gorp.Migration {
	return migrate.New(
		"v63_create_calculations",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			staged, err := scanStaged(tx)
			if err != nil {
				return err
			}
			keys := make(channel.Keys, len(staged))
			for i, st := range staged {
				keys[i] = st.output
			}
			var channels []channel.Channel
			if err = s.channels.NewRetrieve().
				Where(channel.MatchKeys(keys...)).
				Entries(&channels).
				Exec(ctx, tx); err != nil {
				return err
			}
			byKey := make(map[channel.Key]channel.Channel, len(channels))
			for _, ch := range channels {
				byKey[ch.Key()] = ch
			}
			owners := make([]resource.ID, len(staged))
			var warnings []Status
			for i, st := range staged {
				owners[i] = channel.ResourceID(st.output)
				output := byKey[st.output]
				c, dropped := migratedCalculation(st, issuedAtOpen)
				if err = s.table.NewCreate().Entry(&c).Exec(ctx, tx); err != nil {
					return err
				}
				if err = tx.Delete(ctx, st.key); err != nil {
					return err
				}
				if dropped {
					warnings = append(warnings, droppedStatus(output.Name))
				}
			}
			w := statuses.NewWriter(tx)
			if err := w.DeleteOwned(ctx, owners...); err != nil {
				return err
			}
			return w.SetMany(ctx, &warnings)
		},
	)
}

// migratedOperations returns the record form of the staged operations ops. It drops a
// none operation and an operation with no type, and clears each reset channel that the
// Core never issued. It reports whether it dropped an operation with no type.
func migratedOperations(
	ops []channelv2.Operation,
	issuedAtOpen func(channel.Key) bool,
) ([]Operation, bool) {
	var (
		out     = make([]Operation, 0, len(ops))
		dropped bool
	)
	for _, op := range ops {
		w := Window{Duration: op.Duration}
		if op.ResetChannel != 0 && issuedAtOpen(op.ResetChannel) {
			w.Reset = new(op.ResetChannel)
		}
		var variant OperationVariant
		switch op.Type {
		case channelv2.OperationTypeMin:
			variant = MinOperation{Window: w}
		case channelv2.OperationTypeMax:
			variant = MaxOperation{Window: w}
		case channelv2.OperationTypeAvg:
			variant = AvgOperation{Window: w}
		case channelv2.OperationTypeDerivative:
			variant = DerivativeOperation{}
		case channelv2.OperationTypeNone:
			continue
		default:
			dropped = true
			continue
		}
		out = append(out, Operation{Variant: variant})
	}
	return out, dropped
}

// droppedStatus returns the warning that the upgrade dropped operations of the channel
// named name. It has no owner, since the graph writes the status of a calculation.
func droppedStatus(name string) Status {
	return Status{
		Name:    name,
		Variant: status.VariantWarning,
		Message: "Dropped operations with no type",
		Description: fmt.Sprintf(
			"The upgrade dropped the stored operations of %s that had no type. Add "+
				"them again to keep them.",
			name,
		),
		Time: telem.Now(),
	}
}

// staged is a calculation that the channel migrations staged under key for the channel
// output.
type staged struct {
	channelv2.StagedCalculation
	key    []byte
	output channel.Key
}

// scanStaged returns every staged calculation. It reads them all before the migration
// writes, since writes while iterating the staging prefix are unsafe.
func scanStaged(tx gorp.Tx) (out []staged, err error) {
	iter, err := tx.OpenIterator(kv.IterPrefix([]byte(channelv2.CalculationKVPrefix)))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		st := staged{key: bytes.Clone(iter.Key())}
		suffix := strings.TrimPrefix(string(st.key), channelv2.CalculationKVPrefix)
		output, err := strconv.ParseUint(suffix, 10, 32)
		if err != nil {
			return nil, err
		}
		st.output = channel.Key(output)
		if err = json.Unmarshal(iter.Value(), &st.StagedCalculation); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, iter.Error()
}

// migratedCalculation returns the calculation that st stages, and whether it dropped
// an operation of no type. Its reads stay empty until the graph's start compile.
func migratedCalculation(
	st staged,
	issuedAtOpen func(channel.Key) bool,
) (Calculation, bool) {
	ops, dropped := migratedOperations(st.Operations, issuedAtOpen)
	return Calculation{
		Key:        migratedKey(st.output),
		Output:     st.output,
		Expression: st.Expression,
		Operations: ops,
	}, dropped
}
