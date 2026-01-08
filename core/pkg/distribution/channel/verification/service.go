// Copyright 2026 Synnax Labs, Inc.
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
	"fmt"
	"io"
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

const FreeCount = 50

// ServiceConfig is the configuration for a verification service
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
	// DB is the database used for storing the verifier.
	//
	// [REQUIRED]
	kv.DB
	// Verifier is the verifier used for verifying things that need to be verified.
	//
	// [OPTIONAL] - Defaults to ""
	Verifier string
	// CheckInterval is the interval at which verification will be performed.
	//
	// [OPTIONAL] - Defaults to 24 hours
	CheckInterval time.Duration
	// WarningTime is the period given to start warning before verification will fail.
	//
	// [OPTIONAL] - Defaults to 1 week
	WarningTime time.Duration
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

var retrieveKey = []byte("bGljZW5zZUtleQ==")

// Validate validates the configuration for use in the service.
func (c ServiceConfig) Validate() error {
	v := validate.New("channel.verification")
	validate.NotNil(v, "db", c.DB)
	validate.NonZero(v, "warning_time", c.WarningTime)
	validate.NonZero(v, "check_interval", c.CheckInterval)
	return v.Error()
}

// Override replaces fields on c with valid fields from other.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
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
var DefaultConfig = ServiceConfig{
	CheckInterval: 24 * time.Hour,
	WarningTime:   7 * 24 * time.Hour,
}

// Service provides a service for verifying channels.
type Service struct {
	info
	shutdown io.Closer
	cfg      ServiceConfig
}

var _ io.Closer = &Service{}

var (
	useFreeLog = fmt.Sprintf(
		base64.MustDecode(
			"dXNpbmcgU3lubmF4IHdpdGhvdXQgYSBsaWNlbnNlIGtleSwgdXNhZ2UgaXMgbGltaXRlZCB0byAlZCBjaGFubmVscw==",
		),
		FreeCount,
	)
	newRegisteredTemplate = base64.MustDecode(
		"bmV3IGxpY2Vuc2Uga2V5IHJlZ2lzdGVyZWQsIGxpbWl0IGlzICVkIGNoYW5uZWxz",
	)
)

// OpenService opens a new verification service.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	service := &Service{cfg: cfg}
	sCtx, cancel := signal.Isolated(
		signal.WithInstrumentation(service.cfg.Instrumentation),
	)
	service.shutdown = signal.NewHardShutdown(sCtx, cancel)

	startLogMonitor := func() {
		sCtx.Go(
			service.log,
			signal.WithRetryOnPanic(),
			signal.WithBaseRetryInterval(2*time.Second),
			signal.WithRetryScale(1.1),
			signal.WithKey("verification"),
		)
	}

	if cfg.Verifier == "" {
		if err = service.loadCache(ctx); err != nil {
			if !errors.Is(err, kv.NotFound) {
				return nil, err
			}
			cfg.L.Info(useFreeLog)
			return service, nil
		}
		startLogMonitor()
		return service, nil
	}

	if err = service.create(ctx, cfg.Verifier); err != nil {
		return nil, err
	}
	cfg.L.Infof(newRegisteredTemplate, service.numCh)
	startLogMonitor()
	return service, nil
}

// Close should be called when the service is no longer needed.
func (s *Service) Close() error { return s.shutdown.Close() }

// IsOverflowed tells if inUse causes the service to overflow.
func (s *Service) IsOverflowed(inUse types.Uint20) error {
	if s.numCh == 0 {
		if inUse > FreeCount {
			return ErrFree
		}
		return nil
	}
	if s.exprTime.Before(time.Now()) {
		if inUse > FreeCount {
			return ErrStale
		}
		return nil
	}
	if inUse > s.numCh {
		return newErrTooMany(s.numCh)
	}
	return nil
}

var (
	expiredLog = fmt.Sprintf(base64.MustDecode(
		"ZXhwaXJlZCBsaWNlbnNlIGtleSBmb3VuZCwgdXNhZ2UgaXMgbGltaXRlZCB0byAlZCBjaGFubmVscw==",
	), FreeCount)
	existingLogTemplate = base64.MustDecode(
		"ZXhpc3RpbmcgbGljZW5zZSBrZXkgZm91bmQsIHVzYWdlIGlzIGxpbWl0ZWQgdG8gJWQgY2hhbm5lbHM=",
	)
)

func (s *Service) loadCache(ctx context.Context) error {
	key, closer, err := s.cfg.Get(ctx, retrieveKey)
	if err != nil {
		return err
	}
	if err = closer.Close(); err != nil {
		return err
	}
	licenseInf, err := parse(string(key))
	if err != nil {
		return err
	}
	if licenseInf.exprTime.Before(time.Now()) {
		s.cfg.L.Warn(expiredLog)
	} else {
		s.cfg.L.Infof(existingLogTemplate, licenseInf.numCh)
	}
	s.info = licenseInf
	return nil
}

func (s *Service) create(ctx context.Context, toCreate string) error {
	licenseInf, err := parse(toCreate)
	if err != nil {
		return err
	}
	if err = s.cfg.Set(ctx, retrieveKey, []byte(toCreate)); err != nil {
		return err
	}
	s.info = licenseInf
	return nil
}

var (
	hadExpiredLog = fmt.Sprintf(
		base64.MustDecode(
			"bGljZW5zZSBrZXkgZXhwaXJlZCwgdXNhZ2UgaXMgbGltaXRlZCB0byAlZCBjaGFubmVscy4=",
		),
		FreeCount,
	)
	willExpireLogTemplate = base64.MustDecode(
		"bGljZW5zZSBrZXkgd2lsbCBleHBpcmUgaW4gJXM=",
	)
)

func (s *Service) log(ctx context.Context) error {
	if s.exprTime.IsZero() {
		return nil
	}
	ticker := time.NewTicker(s.cfg.CheckInterval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if s.exprTime.Before(time.Now()) {
				s.cfg.L.Error(
					hadExpiredLog,
					zap.String("expired_at", s.exprTime.String()),
				)
			} else if timeLeft := time.Until(
				s.exprTime,
			); timeLeft <= s.cfg.WarningTime {
				s.cfg.L.Warn(
					fmt.Sprintf(willExpireLogTemplate, timeLeft),
					zap.String("expires_in", timeLeft.String()),
				)
			}
		}
	}
}
