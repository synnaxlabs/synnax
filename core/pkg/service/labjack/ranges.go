// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package labjack

import (
	"context"
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// maxRanges are the largest analog input ranges, in Volts, of the models whose ranges a
// channel selects, keyed by device model. A T4 has fixed ranges, so it is absent.
var maxRanges = map[string]float64{"LJM_dtT7": 10, "LJM_dtT8": 11}

// maxRangeOf returns the largest analog input range of the model of c's device, and the
// model. ok is false when c has no device, the device does not exist, or its model has
// fixed ranges.
func maxRangeOf(
	ctx context.Context,
	snap gorp.Snapshot,
	devices *device.Service,
	c ReadConfig,
) (limit float64, model string, ok bool, err error) {
	if c.Device == "" {
		return 0, "", false, nil
	}
	var found []device.Device
	if err = devices.NewRetrieve().
		Where(device.MatchKeys(c.Device)).
		SkipMissingKeys().
		Entries(&found).
		Exec(ctx, snap); err != nil || len(found) == 0 {
		return 0, "", false, err
	}
	limit, ok = maxRanges[found[0].Model]
	return limit, strings.TrimPrefix(found[0].Model, "LJM_dt"), ok, nil
}

// validateRanges rejects an analog channel whose range is above the largest range of
// the model of c's device.
func validateRanges(
	ctx context.Context,
	snap gorp.Snapshot,
	devices *device.Service,
	c ReadConfig,
) error {
	limit, model, ok, err := maxRangeOf(ctx, snap, devices, c)
	if err != nil || !ok {
		return err
	}
	v := validate.New("labjack.ReadConfig")
	for i, ch := range c.Channels {
		a, ok := ch.Variant.(AnalogReadChannel)
		if !ok || a.Range <= limit {
			continue
		}
		v.Exec(func() error {
			return validate.PathedError(
				validate.Errorf(
					"%v V is above the largest range of a %s, %v V",
					a.Range,
					model,
					limit,
				),
				"channels", strconv.Itoa(i), "range",
			)
		})
	}
	return v.Error()
}

// reconcileRanges lowers an analog channel whose range is above the largest range of
// the model of c's device to that range. It returns the indices of the channels it
// changes.
func reconcileRanges(
	ctx context.Context,
	tx gorp.Tx,
	devices *device.Service,
	c *ReadConfig,
) ([]int, error) {
	limit, _, ok, err := maxRangeOf(ctx, tx, devices, *c)
	if err != nil || !ok {
		return nil, err
	}
	var changed []int
	for i := range c.Channels {
		if a, ok := c.Channels[i].Variant.(AnalogReadChannel); ok && a.Range > limit {
			a.Range = limit
			c.Channels[i].Variant = a
			changed = append(changed, i)
		}
	}
	return changed, nil
}

// readModelRangeMigration lowers the ranges of the stored read configs as
// reconcileRanges does, and logs each channel it changes.
func readModelRangeMigration(devices *device.Service) gorp.Migration {
	return migrate.New(
		"v63_labjack_read_model_ranges",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var configs []ReadConfig
			if err := gorp.NewRetrieve[uuid.UUID, ReadConfig]().
				Entries(&configs).
				Exec(ctx, tx); err != nil {
				return err
			}
			for _, c := range configs {
				changed, err := reconcileRanges(ctx, tx, devices, &c)
				if err != nil {
					return err
				}
				if len(changed) == 0 {
					continue
				}
				for _, i := range changed {
					ins.L.Warn(
						"lowered a LabJack range to the largest range of its model",
						zap.Stringer("config", c.Key),
						zap.String("channel", c.Channels[i].Base().Key),
					)
				}
				if err = gorp.NewCreate[uuid.UUID, ReadConfig]().
					Entry(&c).
					Exec(ctx, tx); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
