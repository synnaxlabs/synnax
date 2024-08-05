// Copyright 2024 Synnax Labs, Inc.
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
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type Config struct {
	DB            kv.DB
	Ins           alamos.Instrumentation
	WarningTime   time.Duration
	CheckInterval time.Duration
}

var (
	DefaultConfig = Config{
		WarningTime:   7 * 24 * time.Hour,
		CheckInterval: 24 * time.Hour,
	}
	errStale = errors.New(decode("dXNpbmcgYW4gZXhwaXJlZCBwcm9kdWN0IGxpY2Vuc2Uga2V5LCB1c2UgaXMgbGltaXRlZCB0byB0aGUgZmlyc3Qg") +
		strconv.Itoa(freeCount) + decode("IGNoYW5uZWxz"))
	errFree = errors.New(decode("dXNpbmcgbW9yZSB0aGFuIA==") + strconv.Itoa(freeCount) +
		decode("IGNoYW5uZWxzIHdpdGhvdXQgYSBwcm9kdWN0IGxpY2Vuc2Uga2V5"))
	useFree = decode("dXNpbmcgdGhlIGZyZWUgdmVyc2lvbiBvZiBTeW5uYXgsIG9ubHkg") +
		strconv.Itoa(freeCount) + decode("IGNoYW5uZWxzIGFyZSBhbGxvd2Vk")
)

func (c Config) Validate() error {
	v := validate.New("key")
	validate.NotNil(v, "DB", c.DB)
	validate.NonZero(v, decode("V2FybmluZ1RpbWU="), c.WarningTime)
	validate.NonZero(v, decode("Q2hlY2tJbnRlcnZhbA=="), c.CheckInterval)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ins = override.Zero(c.Ins, other.Ins)
	c.WarningTime = override.If(c.WarningTime, other.WarningTime,
		other.WarningTime.Nanoseconds() != 0)
	return c
}

type Service struct {
	Config
	shutdown io.Closer
}

func OpenService(toOpen string, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	service := &Service{Config: cfg}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(service.Ins))
	service.shutdown = signal.NewShutdown(sCtx, cancel)
	var ctx context.Context
	if toOpen == "" {
		_, err := service.retrieve(ctx)
		if err != nil {
			service.Ins.L.Info(useFree)
			return service, nil
		}
		service.Ins.L.Info(decode("dXNpbmcgdGhlIGxhc3QgbGljZW5zZSBrZXkgc3RvcmVkIGluIHRoZSBkYXRhYmFzZQ=="))
		sCtx.Go(
			service.logTheDog,
			signal.WithRetryOnPanic(),
			signal.WithBaseRetryInterval(2*time.Second),
			signal.WithRetryScale(1.1),
		)
		return service, nil
	}
	err = service.create(ctx, toOpen)
	if err != nil {
		return service, err
	}
	sCtx.Go(
		service.logTheDog,
		signal.WithRetryOnPanic(),
		signal.WithBaseRetryInterval(2*time.Second),
		signal.WithRetryScale(1.1),
	)
	service.Ins.L.Info(decode("bmV3IGxpY2Vuc2Uga2V5IHJlZ2lzdGVyZWQsIGxpbWl0IGlzIA==") +
		strconv.Itoa(int(getNumChan(toOpen))) + decode("IGNoYW5uZWxz"))
	return service, err
}

func (s *Service) Close() error {
	return s.shutdown.Close()
}

func (s *Service) IsOverflowed(ctx context.Context, inUse types.Uint20) error {
	key, err := s.retrieve(ctx)
	if err != nil {
		if inUse > freeCount {
			return errFree
		}
		return nil
	}
	if whenStale(key).Before(time.Now()) {
		if inUse > freeCount {
			return errStale
		}
		return nil
	}
	if channelsAllowed := getNumChan(key); inUse > channelsAllowed {
		return errTooMany(int(channelsAllowed))
	}
	return nil
}

func (s *Service) create(ctx context.Context, toCreate string) error {
	err := validateInput(toCreate)
	if err != nil {
		return err
	}
	return s.DB.Set(ctx, []byte("bGljZW5zZUtleQ=="), []byte(toCreate))
}

func (s *Service) retrieve(ctx context.Context) (string, error) {
	key, err := s.DB.Get(ctx, []byte("bGljZW5zZUtleQ=="))
	return string(key), err
}

func (s *Service) logTheDog(ctx context.Context) error {
	key, err := s.retrieve(ctx)
	if err != nil {
		return err
	}
	staleTime := whenStale(key)
	ticker := time.NewTicker(s.CheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if staleTime.Before(time.Now()) {
				s.Ins.L.Error(decode("TGljZW5zZSBrZXkgZXhwaXJlZC4gQWNjZXNzIGhhcyBiZWVuIGxpbWl0ZWQu"),
					zap.String("ZXhwaXJlZEF0", staleTime.String()))
			} else if timeLeft := time.Until(staleTime); timeLeft <= s.WarningTime {
				s.Ins.L.Warn(decode("TGljZW5zZSBrZXkgd2lsbCBleHBpcmUgc29vbi4gQWNjZXNzIHdpbGwgYmUgbGltaXRlZC4="),
					zap.String(decode("ZXhwaXJlc0lu"), timeLeft.String()))
			} else {
				s.Ins.L.Info(decode("TGljZW5zZSBrZXkgaXMgbm90IGV4cGlyZWQu"),
					zap.String(decode("ZXhwaXJlc0lu"), timeLeft.String()))
			}
		}
	}
}

func errTooMany(count int) error {
	msg := decode("dHJ5aW5nIHRvIHVzZSBtb3JlIHRoYW4gdGhlIGxpbWl0IG9mIA==") +
		strconv.Itoa(count) + decode("IGNoYW5uZWxzIGFsbG93ZWQ=")
	return errors.New(msg)
}
