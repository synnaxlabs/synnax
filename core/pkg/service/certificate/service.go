// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"context"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// DB stores certificates.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Secret stores the private key of each certificate that has one.
	//
	// [REQUIRED]
	Secret *secret.Service
	// Status holds the expiry status of each certificate.
	//
	// [REQUIRED]
	Status *status.Service
	// Signals publishes changes to certificates.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// Now returns the current time.
	//
	// [OPTIONAL] - Defaults to telem.Now.
	Now func() telem.TimeStamp
	alamos.Instrumentation
	// ExpiryCheckInterval is how often the Service updates the expiry status of every
	// certificate.
	//
	// [OPTIONAL] - Defaults to one hour.
	ExpiryCheckInterval telem.TimeSpan
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a Service.
	DefaultServiceConfig = ServiceConfig{
		ExpiryCheckInterval: telem.Hour,
		Now:                 telem.Now,
	}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Secret = override.Nil(c.Secret, other.Secret)
	c.Status = override.Nil(c.Status, other.Status)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Now = override.Nil(c.Now, other.Now)
	c.ExpiryCheckInterval = override.Numeric(
		c.ExpiryCheckInterval,
		other.ExpiryCheckInterval,
	)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("certificate.service")
	v.NotNil("db", c.DB)
	v.NotNil("secret", c.Secret)
	v.NotNil("status", c.Status)
	v.NotNil("now", c.Now)
	v.Positive("expiry_check_interval", c.ExpiryCheckInterval)
	return v.Error()
}

// Service stores certificates, reads their details, and reports their expiry as
// statuses.
type Service struct {
	table   *gorp.Table[Key, Certificate]
	state   *actions.State[Key, Action]
	indexes indexes
	closer  xio.MultiCloser
	cfg     ServiceConfig
}

// OpenService opens a Service with the given configuration. The Service updates the
// expiry status of every certificate when it opens and on each ExpiryCheckInterval. If
// the error is nil, the Service must be closed with Close after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	s = &Service{indexes: newIndexes()}
	if s.cfg, err = config.New(DefaultServiceConfig, cfgs...); err != nil {
		return nil, err
	}
	s.state = actions.NewState[Key, Action](s.cfg.DB)
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Certificate]{
		DB:              s.cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Instrumentation: s.cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if s.cfg.Signals != nil {
		if err = s.cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, Certificate]{
			Changes:  s.table.Changes,
			Resource: ResourceID,
		}); !ok(err, nil) {
			return nil, err
		}
		if err = s.cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
			Name:     "sy_certificate_action_set",
			Actions:  s.state,
			Resource: ResourceID,
		}); !ok(err, nil) {
			return nil, err
		}
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.cfg.Instrumentation))
	if !ok(nil, signal.NewHardShutdown(sCtx, cancel)) {
		return nil, ctx.Err()
	}
	sCtx.Go(s.checkExpiry, signal.WithKey("expiry"))
	return s, nil
}

func (s *Service) checkExpiry(ctx context.Context) error {
	ticker := time.NewTicker(s.cfg.ExpiryCheckInterval.Duration())
	defer ticker.Stop()
	for {
		if err := s.writeExpiry(ctx); err != nil {
			s.cfg.L.Error("failed to update certificate expiry", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Close stops the expiry checks and releases the resources of the Service. Close is not
// safe to call concurrently with any other method of the Service.
func (s *Service) Close() error { return s.closer.Close() }

// Changes returns the changes that the commit c made to certificates.
func (s *Service) Changes(
	ctx context.Context,
	c gorp.Commit,
) gorp.TxReader[Key, Certificate] {
	return s.table.Changes(ctx, c)
}

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// NewWriter opens a Writer that creates, changes, and deletes certificates in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		secrets:    s.cfg.Secret.NewWriter(tx),
		statuses:   s.cfg.Status.NewWriter(tx),
		now:        s.cfg.Now,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a query that retrieves certificates.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}
