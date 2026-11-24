// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	v8 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v8"
	v9 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v9"
)

// DecodeV13 decodes env as a v13 Schematic when env holds a file of v13 or older,
// Console files included, and returns false for a newer file. The v13 shape holds the
// state lists and colors that v14 no longer stores.
func DecodeV13(ctx context.Context, env imex.Envelope) (v13.Schematic, bool, error) {
	if env.Version > v13.Version {
		return v13.Schematic{}, false, nil
	}
	var (
		from12 = v13.MigrateSchematic
		from11 = then(v12.MigrateSchematic, from12)
		from10 = then(v11.MigrateSchematic, from11)
		from9  = then(v10.MigrateSchematic, from10)
		from8  = then(v9.MigrateSchematic, from9)
		from7  = then(v8.MigrateSchematic, from8)
		s      v13.Schematic
		err    error
	)
	switch env.Version {
	case v13.Version:
		s, err = env.Decode[v13.Schematic]()
	case v12.Version:
		s, err = lift(ctx, env, from12)
	case v11.Version:
		s, err = lift(ctx, env, from11)
	case v10.Version:
		s, err = lift(ctx, env, from10)
	case v9.Version:
		s, err = lift(ctx, env, from9)
	case v8.Version:
		s, err = lift(ctx, env, from8)
	case v7.Version:
		s, err = lift(ctx, env, from7)
	default:
		var s7 v7.Schematic
		if s7, err = decodeConsole(ctx, env); err == nil {
			s, err = from7(ctx, s7)
		}
	}
	if err != nil {
		return v13.Schematic{}, false, err
	}
	return s, true, nil
}

type to13[T any] = func(context.Context, T) (v13.Schematic, error)

// then returns the lift to v13 that runs step and then rest.
func then[A, B any](step func(context.Context, A) (B, error), rest to13[B]) to13[A] {
	return func(ctx context.Context, a A) (v13.Schematic, error) {
		b, err := step(ctx, a)
		if err != nil {
			return v13.Schematic{}, err
		}
		return rest(ctx, b)
	}
}

// lift decodes env as T and lifts it to v13.
func lift[T any](
	ctx context.Context,
	env imex.Envelope,
	up to13[T],
) (v13.Schematic, error) {
	t, err := env.Decode[T]()
	if err != nil {
		return v13.Schematic{}, err
	}
	return up(ctx, t)
}
