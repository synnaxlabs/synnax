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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	taskv3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// MigrateScanConfig lifts a v2 scan config to v3.
func MigrateScanConfig(ctx context.Context, old v2.ScanConfig) (ScanConfig, error) {
	return autoMigrateScanConfig(ctx, old)
}

// ScanMigration lifts stored scan configs to v3.
var ScanMigration = migrate.NewEntry("v61_drop_ignored_models", MigrateScanConfig)

// narrow returns v as a T. It returns an error when v is outside the range of T.
func narrow[T uint8 | uint16](name string, v int32) (T, error) {
	if narrowed := T(v); int32(narrowed) == v {
		return narrowed, nil
	}
	return 0, errors.Wrapf(
		validate.ErrValidation, "%s %d is outside the range of %T", name, v, T(0),
	)
}

// MigrateCJC lifts a v2 cold-junction source. It returns an error when the source port
// is outside the range of a port.
func MigrateCJC(ctx context.Context, old v2.CJC) (CJC, error) {
	cjc, err := autoMigrateCJC(ctx, old)
	if err != nil {
		return CJC{}, err
	}
	if v, ok := old.Variant.(v2.ChanCJC); ok {
		port, err := narrow[uint16]("cjc port", v.Port)
		if err != nil {
			return CJC{}, err
		}
		cjc.Variant = ChanCJC{Port: port}
	}
	return cjc, nil
}

// MigrateBaseAIChannel lifts the fields every v2 analog input channel shares. It
// returns an error when the port is outside the range of a port.
func MigrateBaseAIChannel(
	ctx context.Context,
	old v2.BaseAIChannel,
) (BaseAIChannel, error) {
	ch, err := autoMigrateBaseAIChannel(ctx, old)
	if err != nil {
		return BaseAIChannel{}, err
	}
	if ch.Port, err = narrow[uint16]("port", old.Port); err != nil {
		return BaseAIChannel{}, err
	}
	return ch, nil
}

// MigrateBaseCIChannel lifts the fields every v2 counter input channel shares. It
// returns an error when the port is outside the range of a port.
func MigrateBaseCIChannel(
	ctx context.Context,
	old v2.BaseCIChannel,
) (BaseCIChannel, error) {
	ch, err := autoMigrateBaseCIChannel(ctx, old)
	if err != nil {
		return BaseCIChannel{}, err
	}
	if ch.Port, err = narrow[uint16]("port", old.Port); err != nil {
		return BaseCIChannel{}, err
	}
	return ch, nil
}

// MigrateBaseAOChannel lifts the fields every v2 analog output channel shares. It
// returns an error when the port is outside the range of a port.
func MigrateBaseAOChannel(
	ctx context.Context,
	old v2.BaseAOChannel,
) (BaseAOChannel, error) {
	ch, err := autoMigrateBaseAOChannel(ctx, old)
	if err != nil {
		return BaseAOChannel{}, err
	}
	if ch.Port, err = narrow[uint16]("port", old.Port); err != nil {
		return BaseAOChannel{}, err
	}
	return ch, nil
}

// MigrateDIChannel lifts a v2 digital input channel. It returns an error when the port
// or line is outside its range.
func MigrateDIChannel(ctx context.Context, old v2.DIChannel) (DIChannel, error) {
	ch, err := autoMigrateDIChannel(ctx, old)
	if err != nil {
		return DIChannel{}, err
	}
	if ch.Port, err = narrow[uint16]("port", old.Port); err != nil {
		return DIChannel{}, err
	}
	if ch.Line, err = narrow[uint8]("line", old.Line); err != nil {
		return DIChannel{}, err
	}
	return ch, nil
}

// MigrateDOChannel lifts a v2 digital output channel. It returns an error when the port
// or line is outside its range.
func MigrateDOChannel(ctx context.Context, old v2.DOChannel) (DOChannel, error) {
	ch, err := autoMigrateDOChannel(ctx, old)
	if err != nil {
		return DOChannel{}, err
	}
	if ch.Port, err = narrow[uint16]("port", old.Port); err != nil {
		return DOChannel{}, err
	}
	if ch.Line, err = narrow[uint8]("line", old.Line); err != nil {
		return DOChannel{}, err
	}
	return ch, nil
}

// MigrateAnalogReadConfig lifts a v2 analog read config to v3.
func MigrateAnalogReadConfig(
	ctx context.Context,
	old v2.AnalogReadConfig,
) (AnalogReadConfig, error) {
	return autoMigrateAnalogReadConfig(ctx, old)
}

// MigrateCounterReadConfig lifts a v2 counter read config to v3.
func MigrateCounterReadConfig(
	ctx context.Context,
	old v2.CounterReadConfig,
) (CounterReadConfig, error) {
	return autoMigrateCounterReadConfig(ctx, old)
}

// MigrateAnalogWriteConfig lifts a v2 analog write config to v3.
func MigrateAnalogWriteConfig(
	ctx context.Context,
	old v2.AnalogWriteConfig,
) (AnalogWriteConfig, error) {
	return autoMigrateAnalogWriteConfig(ctx, old)
}

// MigrateDigitalReadConfig lifts a v2 digital read config to v3.
func MigrateDigitalReadConfig(
	ctx context.Context,
	old v2.DigitalReadConfig,
) (DigitalReadConfig, error) {
	return autoMigrateDigitalReadConfig(ctx, old)
}

// MigrateDigitalWriteConfig lifts a v2 digital write config to v3.
func MigrateDigitalWriteConfig(
	ctx context.Context,
	old v2.DigitalWriteConfig,
) (DigitalWriteConfig, error) {
	return autoMigrateDigitalWriteConfig(ctx, old)
}

// NewAnalogReadMigration returns the migration that lifts stored analog read configs
// to v3 and quarantines each one it cannot lift.
func NewAnalogReadMigration(otg *ontology.Ontology) gorp.Migration {
	return taskv3.NewRecordMigration(
		"v61_narrow_ni_ai_ports",
		MigrateAnalogReadConfig,
		otg,
	)
}

// NewCounterReadMigration returns the migration that lifts stored counter read configs
// to v3 and quarantines each one it cannot lift.
func NewCounterReadMigration(otg *ontology.Ontology) gorp.Migration {
	return taskv3.NewRecordMigration(
		"v61_narrow_ni_ci_ports",
		MigrateCounterReadConfig,
		otg,
	)
}

// NewAnalogWriteMigration returns the migration that lifts stored analog write configs
// to v3 and quarantines each one it cannot lift.
func NewAnalogWriteMigration(otg *ontology.Ontology) gorp.Migration {
	return taskv3.NewRecordMigration(
		"v61_narrow_ni_ao_ports",
		MigrateAnalogWriteConfig,
		otg,
	)
}

// NewDigitalReadMigration returns the migration that lifts stored digital read configs
// to v3 and quarantines each one it cannot lift.
func NewDigitalReadMigration(otg *ontology.Ontology) gorp.Migration {
	return taskv3.NewRecordMigration(
		"v61_narrow_ni_di_ports",
		MigrateDigitalReadConfig,
		otg,
	)
}

// NewDigitalWriteMigration returns the migration that lifts stored digital write
// configs to v3 and quarantines each one it cannot lift.
func NewDigitalWriteMigration(otg *ontology.Ontology) gorp.Migration {
	return taskv3.NewRecordMigration(
		"v61_narrow_ni_do_ports",
		MigrateDigitalWriteConfig,
		otg,
	)
}
