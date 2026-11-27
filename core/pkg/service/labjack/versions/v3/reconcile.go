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
	"math"
	"strconv"
	"strings"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	"go.uber.org/zap"
)

// ReadReconcileMigration reconciles the channels of the stored read configs as
// ReconcileReadChannels does, and logs each channel it changes. Earlier Cores stored
// such channels.
var ReadReconcileMigration = migrate.NewInstrumentedEntry(
	"v63_labjack_read_reconcile",
	func(
		_ context.Context,
		c ReadConfig,
		ins alamos.Instrumentation,
	) (ReadConfig, error) {
		for _, i := range ReconcileReadChannels(c.Channels) {
			ins.L.Warn(
				"reconciled a LabJack read channel",
				zap.Stringer("config", c.Key),
				zap.String("channel", c.Channels[i].Base().Key),
			)
		}
		return c, nil
	},
)

// PairOf returns the negative channel that pairs with port for a differential reading.
// A T7 pairs an even AIN0 to AIN12 with the next channel, and a Mux80 positive in
// AIN48-55, 64-71, 80-87, 96-103, or 112-119 with the channel 8 above it.
func PairOf(port string) (uint16, bool) {
	digits, ok := strings.CutPrefix(port, "AIN")
	if !ok {
		return 0, false
	}
	pos, err := strconv.Atoi(digits)
	switch {
	case err != nil:
		return 0, false
	case pos >= 0 && pos <= 12 && pos%2 == 0:
		return uint16(pos + 1), true
	case pos >= 48 && pos < 128 && (pos-48)%16 < 8:
		return uint16(pos + 8), true
	}
	return 0, false
}

// reconcileNegChan returns the pair of port for a present negative channel, or ground
// when port has no pair. The bool is true when the result differs from neg.
func reconcileNegChan(port string, neg *uint16) (*uint16, bool) {
	if neg == nil {
		return nil, false
	}
	pair, ok := PairOf(port)
	if !ok {
		return nil, true
	}
	return &pair, *neg != pair
}

// ReconcileReadChannels moves each negative channel to the pair of its port, or to
// ground when the port has no pair. It gives the default range to an analog channel
// whose range is not positive and finite. It returns the indices of the channels it
// changes.
func ReconcileReadChannels(channels []ReadChannel) []int {
	var (
		changed  []int
		defaults AnalogReadChannel
	)
	defaults.SetDefaults()
	for i := range channels {
		var negChanged, rangeChanged bool
		switch ch := channels[i].Variant.(type) {
		case AnalogReadChannel:
			ch.NegChan, negChanged = reconcileNegChan(ch.Port, ch.NegChan)
			if rangeChanged = !(ch.Range > 0) || math.IsInf(ch.Range, 1); rangeChanged {
				ch.Range = defaults.Range
			}
			channels[i].Variant = ch
		case ThermocoupleReadChannel:
			ch.NegChan, negChanged = reconcileNegChan(ch.Port, ch.NegChan)
			channels[i].Variant = ch
		}
		if negChanged || rangeChanged {
			changed = append(changed, i)
		}
	}
	return changed
}
