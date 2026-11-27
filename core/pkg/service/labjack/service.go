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
	"math"
	"strconv"
	"strings"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/versions"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/versions/legacy"
	v3 "github.com/synnaxlabs/synnax/pkg/service/labjack/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the LabJack task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("labjack.service")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service owns the stored configuration records of the LabJack task types.
type Service struct {
	// Read stores labjack_read task configuration records.
	Read *config.Service[ReadConfig]
	// Write stores labjack_write task configuration records.
	Write *config.Service[WriteConfig]
	// Scan stores labjack_scan task configuration records.
	Scan   *config.Service[ScanConfig]
	closer xio.MultiCloser
}

// OpenService opens the LabJack task config service with the provided configuration.
// If error is nil, the service is ready for use and must be closed by calling Close
// to prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Read, err = config.OpenService(
		ctx, config.ServiceConfig[ReadConfig]{
			DB:              cfg.DB,
			References:      readConfigReferences(),
			Instrumentation: cfg.Instrumentation,
			Type:            "labjack_read",
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.ReadConfigLadder,
			Legacy:          &legacy.Read,
			Migrations:      []gorp.Migration{v3.ReadMigration},
			SetEntryKey:     (*ReadConfig).SetKey,
			PatchEntry:      (*ReadConfig).Patch,
			HashEntry:       (*ReadConfig).HashContent,
			ReconcileEntry: func(_ context.Context, _ gorp.Tx, c *ReadConfig) error {
				reconcileReadChannels(c)
				return nil
			},
			ValidateEntry: func(c *ReadConfig) error {
				v := validate.New("labjack.ReadConfig")
				v.Exec(c.Validate)
				v.GreaterThan("stream_rate", c.StreamRate, 0)
				for i, ch := range c.Channels {
					v.Exec(func() error {
						return validate.PathedError(
							validateNegChan(negChanOf(ch)),
							"channels", strconv.Itoa(i), "neg_chan",
						)
					})
				}
				return v.Error()
			},
		},
	); !ok(err, s.Read) {
		return nil, err
	}
	if s.Write, err = config.OpenService(
		ctx, config.ServiceConfig[WriteConfig]{
			DB:              cfg.DB,
			References:      writeConfigReferences(),
			Instrumentation: cfg.Instrumentation,
			Type:            "labjack_write",
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.WriteConfigLadder,
			Legacy:          &legacy.Write,
			SetEntryKey:     (*WriteConfig).SetKey,
			PatchEntry:      (*WriteConfig).Patch,
			HashEntry:       (*WriteConfig).HashContent,
			ValidateEntry:   (*WriteConfig).Validate,
		},
	); !ok(err, s.Write) {
		return nil, err
	}
	if s.Scan, err = config.OpenService(
		ctx, config.ServiceConfig[ScanConfig]{
			DB:              cfg.DB,
			Instrumentation: cfg.Instrumentation,
			Type:            "labjack_scan",
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.ScanConfigLadder,
			Legacy:          &legacy.Scan,
			Migrations:      []gorp.Migration{v3.ScanMigration},
			SetEntryKey:     (*ScanConfig).SetKey,
			PatchEntry:      (*ScanConfig).Patch,
			HashEntry:       (*ScanConfig).HashContent,
			ValidateEntry:   (*ScanConfig).Validate,
		},
	); !ok(err, s.Scan) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store {
	return []config.Store{s.Read, s.Write, s.Scan}
}

// pairOf returns the negative channel that pairs with port for a differential reading.
// A T7 pairs an even AIN0 to AIN12 with the next channel, and a Mux80 positive in
// AIN48-55, 64-71, 80-87, 96-103, or 112-119 with the channel 8 above it.
func pairOf(port string) (uint16, bool) {
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

// validateNegChan rejects a negative channel that is not the pair of port.
func validateNegChan(port string, neg *uint16) error {
	if neg == nil {
		return nil
	}
	pair, ok := pairOf(port)
	if !ok {
		return validate.Errorf("%s has no differential pair", port)
	}
	if *neg != pair {
		return validate.Errorf("%s pairs only with AIN%d", port, pair)
	}
	return nil
}

// negChanOf returns the port and negative channel of a channel that has them.
func negChanOf(ch ReadChannel) (string, *uint16) {
	switch c := ch.Variant.(type) {
	case AnalogReadChannel:
		return c.Port, c.NegChan
	case ThermocoupleReadChannel:
		return c.Port, c.NegChan
	}
	return "", nil
}

// reconcileNegChan returns neg when it is absent or the pair of port. Otherwise it
// returns the pair, or ground when port has no pair.
func reconcileNegChan(port string, neg *uint16) *uint16 {
	if neg == nil || validateNegChan(port, neg) == nil {
		return neg
	}
	if pair, ok := pairOf(port); ok {
		return &pair
	}
	return nil
}

// reconcileReadChannels moves the negative channels of c to the pairs of their ports,
// and gives the default range to an analog channel whose range is not positive and
// finite, so that a config stored before these rules passes validation.
func reconcileReadChannels(c *ReadConfig) {
	var defaults AnalogReadChannel
	defaults.SetDefaults()
	for i := range c.Channels {
		switch ch := c.Channels[i].Variant.(type) {
		case AnalogReadChannel:
			ch.NegChan = reconcileNegChan(ch.Port, ch.NegChan)
			if !(ch.Range > 0) || math.IsInf(ch.Range, 1) {
				ch.Range = defaults.Range
			}
			c.Channels[i].Variant = ch
		case ThermocoupleReadChannel:
			ch.NegChan = reconcileNegChan(ch.Port, ch.NegChan)
			c.Channels[i].Variant = ch
		}
	}
}
