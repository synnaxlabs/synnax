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
// state lists and colors that v14 no longer stores. A scale that is not positive
// becomes 1, as in DecodeImExEnvelope.
func DecodeV13(ctx context.Context, env imex.Envelope) (v13.Schematic, bool, error) {
	if env.Version > v13.Version {
		return v13.Schematic{}, false, nil
	}
	var (
		s   v13.Schematic
		err error
	)
	switch env.Version {
	case v13.Version:
		s, err = env.Decode[v13.Schematic]()
	case v12.Version:
		s, err = lift(ctx, env, v13.MigrateSchematic)
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
	for i := range s.Nodes {
		if s.Nodes[i].Config, err = positiveScale(s.Nodes[i].Config); err != nil {
			return v13.Schematic{}, false, err
		}
	}
	return s, true, nil
}

func from11(ctx context.Context, s v11.Schematic) (v13.Schematic, error) {
	s12, err := v12.MigrateSchematic(ctx, s)
	if err != nil {
		return v13.Schematic{}, err
	}
	return v13.MigrateSchematic(ctx, s12)
}

func from10(ctx context.Context, s v10.Schematic) (v13.Schematic, error) {
	s11, err := v11.MigrateSchematic(ctx, s)
	if err != nil {
		return v13.Schematic{}, err
	}
	return from11(ctx, s11)
}

func from9(ctx context.Context, s v9.Schematic) (v13.Schematic, error) {
	s10, err := v10.MigrateSchematic(ctx, s)
	if err != nil {
		return v13.Schematic{}, err
	}
	return from10(ctx, s10)
}

func from8(ctx context.Context, s v8.Schematic) (v13.Schematic, error) {
	s9, err := v9.MigrateSchematic(ctx, s)
	if err != nil {
		return v13.Schematic{}, err
	}
	return from9(ctx, s9)
}

func from7(ctx context.Context, s v7.Schematic) (v13.Schematic, error) {
	s8, err := v8.MigrateSchematic(ctx, s)
	if err != nil {
		return v13.Schematic{}, err
	}
	return from8(ctx, s8)
}

// lift decodes env as T and lifts it to v13 with up.
func lift[T any](
	ctx context.Context,
	env imex.Envelope,
	up func(context.Context, T) (v13.Schematic, error),
) (v13.Schematic, error) {
	t, err := env.Decode[T]()
	if err != nil {
		return v13.Schematic{}, err
	}
	return up(ctx, t)
}
