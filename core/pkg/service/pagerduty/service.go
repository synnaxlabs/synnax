// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/legacy"
	v2 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	xconfig "github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the PagerDuty task config service.
type ServiceConfig struct {
	// DB is the database config records are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Secret stores the routing key of each alert task.
	// [REQUIRED]
	Secret *secret.Service
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Secret = override.Nil(c.Secret, other.Secret)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("pagerduty.service")
	v.NotNil("db", c.DB)
	v.NotNil("secret", c.Secret)
	return v.Error()
}

// Service owns the stored configuration records of the PagerDuty alert task type.
type Service struct {
	// Alert stores pagerduty_alert task configuration records.
	Alert  *config.Service[TaskConfig]
	closer xio.MultiCloser
}

// OpenService opens the PagerDuty task config service with the provided
// configuration. If error is nil, the service is ready for use and must be closed by
// calling Close to prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	create := createSecret(cfg.Secret)
	s = &Service{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Alert, err = config.OpenService(
		ctx, config.ServiceConfig[TaskConfig]{
			DB:              cfg.DB,
			Instrumentation: cfg.Instrumentation,
			Type:            AlertTaskType,
			LegacyVersion:   legacy.LastVersion,
			Ladder:          versions.TaskConfigLadder,
			Legacy:          &legacy.Alert,
			Migrations:      []gorp.Migration{v2.NewMigration(create)},
			SetEntryKey:     (*TaskConfig).SetKey,
			ResolveEntry: func(
				ctx context.Context,
				tx gorp.Tx,
				name string,
				c *TaskConfig,
			) error {
				return v2.ResolveTaskConfig(ctx, tx, create, name, c)
			},
		},
	); !ok(err, s.Alert) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases the resources it holds. Close is not safe
// to call concurrently with any other service methods.
func (s *Service) Close() error { return s.closer.Close() }

// Stores returns the config stores the service owns, for registry assembly.
func (s *Service) Stores() []config.Store {
	return []config.Store{s.Alert}
}

func createSecret(svc *secret.Service) v2.CreateSecret {
	return func(
		ctx context.Context,
		tx gorp.Tx,
		name string,
		value []byte,
	) (secret.Key, error) {
		s := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{Name: name}}}
		if err := svc.NewWriter(tx).Create(ctx, &s, value); err != nil {
			return uuid.Nil(), err
		}
		return s.Base().Key, nil
	}
}
