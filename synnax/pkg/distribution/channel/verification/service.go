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
	"encoding/base64"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type Config struct {
	// DB is the database used to store information.
	DB kv.DB
	// Ins is the instrumentation used for logging
	Ins alamos.Instrumentation
	// ExpireWarning is the warning time until the key expires
	ExpireWarning time.Duration
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{
		ExpireWarning: 7 * 24 * time.Hour,
	}
)

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("key")
	validate.NotNil(v, "DB", c.DB)
	validate.NonZero(v, "ExpireWarning", c.ExpireWarning)
	return v.Error()
}

// Override implements config.Properties.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ins = override.Zero(c.Ins, other.Ins)
	c.ExpireWarning = override.If(c.ExpireWarning, other.ExpireWarning, other.ExpireWarning.Nanoseconds() != 0)
	return c
}

type Service struct {
	Config
	// shutdown
	shutdown io.Closer
}

// OpenService opens a new service by creating a key from the argument
// key stored in DB.
func OpenService(key string, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	service := &Service{Config: cfg}

	// This stuff needs to be updated when I decide what to do with the
	// background key validation process
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(service.Ins))
	service.shutdown = signal.NewShutdown(sCtx, cancel)

	var ctx context.Context
	if key == "" {
		return service, nil
	}

	// user did enter a key
	err = service.create(ctx, key)
	if err != nil {
		return service, err
	}
	sCtx.Go(service.updateKeyValidation)

	return service, err
}

// Close will shutdown the service
func (s *Service) Close() error {
	return s.shutdown.Close()
}

// ValidateChannelCount returns an error if inUse is greater than the number of
// channels the user is allowed to use.
func (s *Service) ValidateChannelCount(ctx context.Context, inUse int64) error {
	key, err := s.retrieve(ctx)

	if err != nil {
		if inUse > maxFreeChannels {
			return errFreeLimit(maxFreeChannels)
		}
		return nil
	}

	if getExpirationDate(key).Before(time.Now()) {
		if inUse > maxFreeChannels {
			return errExpireKey(maxFreeChannels)
		}
		return nil
	}

	if channelsAllowed := getNumberOfChannels(key); inUse > channelsAllowed {
		return errOverChannelLimit(int(channelsAllowed))
	}

	return nil
}

// IsExpired returns an error if the key stored in s is expired.
func (s *Service) IsExpired(ctx context.Context) error {
	key, err := s.retrieve(ctx)
	if err != nil {
		return errFreeLimit(maxFreeChannels)

	}
	if getExpirationDate(key).Before(time.Now()) {
		return errExpireKey(maxFreeChannels)
	}
	return nil
}

func (s *Service) GetMaxFreeChannels() int {
	return maxFreeChannels
}

// create sets  key in s.DB. Returns an error if the key is invalid - you can
// only set a valid key in the database
func (s *Service) create(ctx context.Context, key string) error {
	err := validateKey(key)
	if err != nil {
		return err
	}
	return s.DB.Set(ctx, []byte("productKey"), []byte(key))
}

// retrieve grabs the key in s.DB.
func (s *Service) retrieve(ctx context.Context) (string, error) {
	key, err := s.DB.Get(ctx, []byte("productKey"))
	return string(key), err
}

// updateKeyValidation will run a background process that continuously checks if
// the key is expired.
func (s *Service) updateKeyValidation(ctx context.Context) error {

	key, err := s.retrieve(ctx)
	if err != nil {
		return err
	}
	expireTime := getExpirationDate(key)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if expireTime.Before(time.Now()) {
				s.Ins.L.Error("License key expired. Access has been limited", zap.String("expiredAt", expireTime.String()))
			} else if timeLeft := time.Until(expireTime); timeLeft <= s.ExpireWarning {
				s.Ins.L.Warn("License key will expire soon. Access will be limited.", zap.String("expiresIn", timeLeft.String()))
			} else {
				s.Ins.L.Info("License key is not expired.", zap.String("expiresIn", timeLeft.String()))
			}
		}
	}
}

func errExpireKey(channelsAllowed int) error {
	encoded1 := "dXNpbmcgYW4gZXhwaXJlZCBwcm9kdWN0IGxpY2Vuc2Uga2V5LCB1c2UgaXMgbGltaXRlZCB0byB0aGUgZmlyc3Qg"
	decoded1, _ := base64.StdEncoding.DecodeString(encoded1)
	encoded2 := "IGNoYW5uZWxz"
	decoded2, _ := base64.StdEncoding.DecodeString(encoded2)

	msg := string(decoded1) + strconv.Itoa(channelsAllowed) + string(decoded2)
	return errors.New(msg)
}

func errOverChannelLimit(chanLimit int) error {
	encoded1 := "dXNpbmcgbW9yZSB0aGFuIA=="
	decoded1, _ := base64.StdEncoding.DecodeString(encoded1)
	encoded2 := "IGNoYW5uZWxzIGFsbG93ZWQ="
	decoded2, _ := base64.StdEncoding.DecodeString(encoded2)

	msg := string(decoded1) + strconv.Itoa(chanLimit) + string(decoded2)
	return errors.New(msg)
}

func errFreeLimit(channelsAllowed int) error {
	encoded1 := "dXNpbmcgbW9yZSB0aGFuIA=="
	decoded1, _ := base64.StdEncoding.DecodeString(encoded1)
	encoded2 := "IGNoYW5uZWxzIHdpdGhvdXQgYSBwcm9kdWN0IGxpY2Vuc2Uga2V5"
	decoded2, _ := base64.StdEncoding.DecodeString(encoded2)
	msg := string(decoded1) + strconv.Itoa(channelsAllowed) + string(decoded2)
	return errors.New(msg)
}
