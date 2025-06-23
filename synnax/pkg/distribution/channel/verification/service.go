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
	freeCount   = 50
)

// Config is the configuration for a verification service
type Config struct {
	// DB is used to persist
	DB            kv.DB
	Verifier      string
	Ins           alamos.Instrumentation
	WarningTime   time.Duration
	CheckInterval time.Duration
}

var (
	DefaultConfig = Config{
		WarningTime: 7 * 24 * time.Hour, CheckInterval: 24 * time.Hour,
	}
	freeCountStr   = strconv.Itoa(freeCount)
	limitErrPrefix = base64.MustDecode("dXNpbmcgbW9yZSB0aGFuIA==")
	errStale       = errors.New(
		limitErrPrefix +
			freeCountStr +
			base64.MustDecode("IGNoYW5uZWxzIHdpdGggYW4gZXhwaXJlZCBsaWNlbnNlIGtleQ=="),
	) // using more than 50 channels with an expired license key
	errFree = errors.New(
		limitErrPrefix + freeCountStr +
			base64.MustDecode("IHdpdGhvdXQgYSBsaWNlbnNlIGtleQ=="),
	) // using more than 50 channels without a license key
	useFree = base64.MustDecode("dXNpbmcgU3lubmF4IHdpdGhvdXQgYSBsaWNlbnNlIGtleSwgdXNhZ2UgaXMgbGltaXRlZCB0byA=") +
		freeCountStr + base64.MustDecode("IGNoYW5uZWxzIGFyZSBhbGxvd2Vk") // using Synnax without a license key, usage is limited to
	usingDBStr = base64.MustDecode("dXNpbmcgdGhlIGxhc3QgbGljZW5zZSBrZXkgc3RvcmVkIGluIHRoZSBkYXRhYmFzZSwgdXNhZ2UgaXMgbGltaXRlZCB0byA=") // using the last license key stored in the database, usage is limited to
	chStr      = base64.MustDecode("IGNoYW5uZWxz")                                                                                     //  channels
)

// Validate implements config.Config
func (c Config) Validate() error {
	v := validate.New("key")
	validate.NotNil(v, "DB", c.DB)
	validate.NonZero(v, "WarningTime", c.WarningTime)
	validate.NonZero(v, "CheckInterval", c.CheckInterval)
	return v.Error()
}

// Override implements config.Config
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ins = override.Zero(c.Ins, other.Ins)
	c.WarningTime = override.If(c.WarningTime, other.WarningTime,
		other.WarningTime.Nanoseconds() != 0)
	c.Verifier = override.String(other.Verifier, c.Verifier)
	return c
}

type Service struct {
	Config
	shutdown    io.Closer
	licenseInfo info
}

func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	service := &Service{Config: cfg}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(service.Ins))
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

	// TODO: check if you are opening with a stale key
				return nil, err
			}
			service.Ins.L.Info(useFree)
			return service, nil
		}
		service.Ins.L.Info(usingDBStr + strconv.Itoa(int(service.licenseInfo.numCh)) + chStr)
		return service, nil
	}

<<<<<<< Updated upstream
	err = service.create(ctx, cfg.Verifier)
	if err != nil {
		return service, err
=======
	service.Ins.L.Info(
		base64.MustDecode("bmV3IGxpY2Vuc2Uga2V5IHJlZ2lzdGVyZWQsIGxpbWl0IGlzIA==") +
	)
	startLogMonitor()
<<<<<<< Updated upstream
	service.Ins.L.Info(decode("bmV3IGxpY2Vuc2Uga2V5IHJlZ2lzdGVyZWQsIGxpbWl0IGlzIA==") +
		strconv.Itoa(int(getNumChan(cfg.Verifier))) + decode("IGNoYW5uZWxz"))
	return service, err
=======
	return service, nil
>>>>>>> Stashed changes
}

// Close implements io.Closer
func (s *Service) Close() error {
	return s.shutdown.Close()
}

func (s *Service) IsOverflowed(inUse types.Uint20) error {
	if s.licenseInfo.numCh == 0 {
		if inUse > freeCount {
			return errFree
		}
		return nil
	}
	if s.licenseInfo.exprTime.Before(time.Now()) {
		if inUse > freeCount {
			return errStale
		}
		return nil
	}
	if inUse > s.licenseInfo.numCh {
		return errTooMany(int(s.licenseInfo.numCh))
	}
	return nil
}

func (s *Service) loadCache(ctx context.Context) error {
	key, closer, err := s.DB.Get(ctx, []byte(retrieveKey))
	if err != nil {
		return err
	}
	if err = closer.Close(); err != nil {
		return err
	}
	keyStr := string(key)
	licenseInf, err := parseLicenseKey(keyStr)
	if err != nil {
		return err
	}
	s.licenseInfo = licenseInf
	return nil
}

func (s *Service) create(ctx context.Context, toCreate string) error {
	licenseInf, err := parseLicenseKey(toCreate)
	if err != nil {
		return err
	}
	if err = s.DB.Set(ctx, []byte(retrieveKey), []byte(toCreate)); err != nil {
		return err
	}
	s.licenseInfo = licenseInf
	return nil
}

func (s *Service) logTheDog(ctx context.Context) error {
	if s.licenseInfo.exprTime.IsZero() {
		return nil
	}
	ticker := time.NewTicker(s.CheckInterval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if s.licenseInfo.exprTime.Before(time.Now()) {
				s.Ins.L.Error(
					base64.MustDecode(
						"TGljZW5zZSBrZXkgZXhwaXJlZC4gQWNjZXNzIGhhcyBiZWVuIGxpbWl0ZWQu",
					),
					zap.String("ZXhwaXJlZEF0", s.licenseInfo.exprTime.String()),
				)
			} else if timeLeft := time.Until(s.licenseInfo.exprTime); timeLeft <= s.WarningTime {
				s.Ins.L.Warn(
					base64.MustDecode(
						"TGljZW5zZSBrZXkgd2lsbCBleHBpcmUgc29vbi4gQWNjZXNzIHdpbGwgYmUgbGltaXRlZC4=",
					),
					zap.String(
						base64.MustDecode("ZXhwaXJlc0lu"), timeLeft.String(),
					),
				)
			} else {
				s.Ins.L.Info(
					base64.MustDecode("TGljZW5zZSBrZXkgaXMgbm90IGV4cGlyZWQu"),
					zap.String(
						base64.MustDecode("ZXhwaXJlc0lu"), timeLeft.String(),
					),
				)
			}
		}
	}
}

func errTooMany(count int) error {
	msg := base64.MustDecode("dHJ5aW5nIHRvIHVzZSBtb3JlIHRoYW4gdGhlIGxpbWl0IG9mIA==") +
		strconv.Itoa(count) + base64.MustDecode("IGNoYW5uZWxzIGFsbG93ZWQ=")
	return errors.New(msg)
}
