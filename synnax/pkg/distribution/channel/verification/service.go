// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/base64"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

const (
	retrieveKey = "bGljZW5zZUtleQ=="
	FreeCount   = 50
)

// Config is the configuration for a verification service
type Config struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
	// DB is the database used for storing the verifier.
	//
	// [REQUIRED]
	kv.DB
	// CheckInterval is the interval at which verification will be performed.
	//
	// [OPTIONAL] - Defaults to 24 hours
	CheckInterval time.Duration
	// Verifier is the verifier used for verifying things that need to be verified.
	//
	// [OPTIONAL] - Defaults to ""
	Verifier string
	// WarningTime is the period given to start warning before verification will fail.
	//
	// [OPTIONAL] - Defaults to 1 week
	WarningTime time.Duration
}

var _ config.Config[Config] = Config{}

var (
	freeCountStr   = strconv.Itoa(FreeCount)
	limitErrPrefix = base64.MustDecode("dXNpbmcgbW9yZSB0aGFuIA==")
	ErrStale       = errors.New(
		limitErrPrefix +
			freeCountStr +
			base64.MustDecode("IGNoYW5uZWxzIHdpdGggYW4gZXhwaXJlZCBsaWNlbnNlIGtleQ=="),
	)
	ErrFree = errors.New(
		limitErrPrefix +
			freeCountStr +
			base64.MustDecode("IHdpdGhvdXQgYSBsaWNlbnNlIGtleQ=="),
	)
	useFree = base64.MustDecode("dXNpbmcgU3lubmF4IHdpdGhvdXQgYSBsaWNlbnNlIGtleSwgdXNhZ2UgaXMgbGltaXRlZCB0byA=") +
		freeCountStr + base64.MustDecode("IGNoYW5uZWxzIGFyZSBhbGxvd2Vk") // using Synnax without a license key, usage is limited to
	usingDBStr = base64.MustDecode("dXNpbmcgdGhlIGxhc3QgbGljZW5zZSBrZXkgc3RvcmVkIGluIHRoZSBkYXRhYmFzZSwgdXNhZ2UgaXMgbGltaXRlZCB0byA=") // using the last license key stored in the database, usage is limited to
	chStr      = base64.MustDecode("IGNoYW5uZWxz")                                                                                     //  channels
)

// Validate validates the configuration for use in the service.
func (c Config) Validate() error {
	v := validate.New("channel.verification")
	validate.NotNil(v, "db", c.DB)
	validate.NonZero(v, "warning_time", c.WarningTime)
	validate.NonZero(v, "check_interval", c.CheckInterval)
	return v.Error()
}

// Override replaces fields on c with valid fields from other.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.CheckInterval = override.If(c.CheckInterval, other.CheckInterval,
		other.CheckInterval.Nanoseconds() != 0)
	c.WarningTime = override.If(c.WarningTime, other.WarningTime,
		other.WarningTime.Nanoseconds() != 0)
	c.Verifier = override.String(other.Verifier, c.Verifier)
	return c
}

// DefaultConfig is the default configuration for the verification service.
var DefaultConfig = Config{
	CheckInterval: 24 * time.Hour,
	WarningTime:   7 * 24 * time.Hour,
}

// Service provides a service for verifying channels.
type Service struct {
	info
	cfg      Config
	shutdown io.Closer
}

var _ io.Closer = &Service{}

func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	service := &Service{cfg: cfg}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(service.cfg.Instrumentation))
	service.shutdown = signal.NewHardShutdown(sCtx, cancel)

	startLogMonitor := func() {
		sCtx.Go(
			service.logTheDog,
			signal.WithRetryOnPanic(),
			signal.WithBaseRetryInterval(2*time.Second),
			signal.WithRetryScale(1.1),
			signal.WithKey("verification"),
		)
	}

	if cfg.Verifier == "" {
		// TODO: ignoring error here
		if err = service.loadCache(ctx); err != nil {
			cfg.Instrumentation.L.Info(useFree)
			return service, nil
		}
		cfg.Instrumentation.L.Info(usingDBStr + strconv.Itoa(int(service.info.numCh)) + chStr)
		startLogMonitor()
		return service, nil
	}

	if err = service.create(ctx, cfg.Verifier); err != nil {
		return service, err
	}
	cfg.Instrumentation.L.Info(base64.MustDecode("bmV3IGxpY2Vuc2Uga2V5IHJlZ2lzdGVyZWQsIGxpbWl0IGlzIA=="))
	startLogMonitor()
	cfg.Instrumentation.L.Info(base64.MustDecode("bmV3IGxpY2Vuc2Uga2V5IHJlZ2lzdGVyZWQsIGxpbWl0IGlzIA==") +
		strconv.Itoa(int(service.info.numCh)) + base64.MustDecode("IGNoYW5uZWxz"))
	return service, err
}

// Close implements io.Closer
func (s *Service) Close() error { return s.shutdown.Close() }

func (s *Service) IsOverflowed(inUse types.Uint20) error {
	if s.info.numCh == 0 {
		if inUse > FreeCount {
			return ErrFree
		}
		return nil
	}
	if s.info.exprTime.Before(time.Now()) {
		if inUse > FreeCount {
			return ErrStale
		}
		return nil
	}
	if inUse > s.info.numCh {
		return errTooMany(s.info.numCh)
	}
	return nil
}

func (s *Service) loadCache(ctx context.Context) error {
	key, closer, err := s.cfg.DB.Get(ctx, []byte(retrieveKey))
	if err != nil {
		return err
	}
	if err = closer.Close(); err != nil {
		return err
	}
	licenseInf, err := parseLicenseKey(string(key))
	if err != nil {
		return err
	}
	s.info = licenseInf
	return nil
}

func (s *Service) create(ctx context.Context, toCreate string) error {
	licenseInf, err := parseLicenseKey(toCreate)
	if err != nil {
		return err
	}
	if err = s.cfg.DB.Set(ctx, []byte(retrieveKey), []byte(toCreate)); err != nil {
		return err
	}
	s.info = licenseInf
	return nil
}

func (s *Service) logTheDog(ctx context.Context) error {
	if s.info.exprTime.IsZero() {
		return nil
	}
	ticker := time.NewTicker(s.cfg.CheckInterval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if s.info.exprTime.Before(time.Now()) {
				s.cfg.Instrumentation.L.Error(
					base64.MustDecode("TGljZW5zZSBrZXkgZXhwaXJlZC4gQWNjZXNzIGhhcyBiZWVuIGxpbWl0ZWQu"),
					zap.String("ZXhwaXJlZEF0", s.info.exprTime.String()),
				)
			} else if timeLeft := time.Until(s.info.exprTime); timeLeft <= s.cfg.WarningTime {
				s.cfg.Instrumentation.L.Warn(
					base64.MustDecode(
						"TGljZW5zZSBrZXkgd2lsbCBleHBpcmUgc29vbi4gQWNjZXNzIHdpbGwgYmUgbGltaXRlZC4=",
					),
					zap.String(base64.MustDecode("ZXhwaXJlc0lu"), timeLeft.String()),
				)
			} else {
				s.cfg.Instrumentation.L.Info(
					base64.MustDecode("TGljZW5zZSBrZXkgaXMgbm90IGV4cGlyZWQu"),
					zap.String(base64.MustDecode("ZXhwaXJlc0lu"), timeLeft.String()),
				)
			}
		}
	}
}

func errTooMany(count types.Uint20) error {
	msg := base64.MustDecode("dHJ5aW5nIHRvIHVzZSBtb3JlIHRoYW4gdGhlIGxpbWl0IG9mIA==") +
		strconv.Itoa(int(count)) + base64.MustDecode("IGNoYW5uZWxzIGFsbG93ZWQ=")
	return errors.New(msg)
}
