// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	"github.com/synnaxlabs/x/errors"
	payload "github.com/synnaxlabs/x/payload/versions/v1"
)

// MigrateWriteConfig lifts a v4 HTTP write task config to v5.
func MigrateWriteConfig(ctx context.Context, old v4.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

// MigrateWriteEndpoint lifts a v4 write endpoint to v5.
func MigrateWriteEndpoint(
	ctx context.Context,
	old v4.WriteEndpoint,
) (WriteEndpoint, error) {
	return autoMigrateWriteEndpoint(ctx, old)
}

// MigrateWriteField lifts a v4 write field to a typed write field, as
// payload.LiftStaticWriteField and payload.LiftGeneratedWriteField do. ok is false for
// a field with an unknown generator, which no Driver ran, and for a null field.
func MigrateWriteField(old v4.WriteField) (f payload.WriteField, ok bool, err error) {
	switch v := old.Variant.(type) {
	case nil:
		return payload.WriteField{}, false, nil
	case v4.StaticWriteField:
		f, err = payload.LiftStaticWriteField(
			payload.BaseWriteField(v.BaseWriteField),
			v.Value,
		)
		return f, err == nil, err
	case v4.GeneratedWriteField:
		f, ok = payload.LiftGeneratedWriteField(
			payload.BaseWriteField(v.BaseWriteField),
			v.Generator,
			v.TimeFormat,
		)
		return f, ok, nil
	default:
		return payload.WriteField{}, false, errors.Newf(
			"unknown WriteField variant %T",
			v,
		)
	}
}

func migrateWriteEndpointFields(
	_ context.Context,
	old []v4.WriteField,
) ([]payload.WriteField, error) {
	var fields []payload.WriteField
	for _, o := range old {
		f, ok, err := MigrateWriteField(o)
		if err != nil {
			return nil, err
		}
		if ok {
			fields = append(fields, f)
		}
	}
	return fields, nil
}

// WriteMigration lifts stored HTTP write task configs from v4 to v5.
var WriteMigration = migrate.NewEntry("v62_http_typed_write_fields", MigrateWriteConfig)
