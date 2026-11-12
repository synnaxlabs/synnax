// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"
	"crypto/cipher"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// DB stores secrets.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Signals publishes changes to secrets.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// Grant stores the grants that let a rack read secrets.
	//
	// [REQUIRED]
	Grant *grant.Service
	// Now returns the current time.
	//
	// [OPTIONAL] - Defaults to telem.Now.
	Now func() telem.TimeStamp
	alamos.Instrumentation
	// EncryptionKey protects the value of every stored secret. It encrypts the random
	// data key that encrypts each value. Every Core in a cluster must use the same key.
	//
	// [REQUIRED]
	EncryptionKey EncryptionKey
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a Service.
	DefaultServiceConfig = ServiceConfig{Now: telem.Now}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Grant = override.Nil(c.Grant, other.Grant)
	c.Now = override.Nil(c.Now, other.Now)
	c.EncryptionKey = override.If(
		c.EncryptionKey,
		other.EncryptionKey,
		other.EncryptionKey != EncryptionKey{},
	)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("secret.service")
	v.NotNil("db", c.DB)
	v.NotNil("grant", c.Grant)
	v.NotNil("now", c.Now)
	v.Ternary("encryption_key", c.EncryptionKey == EncryptionKey{}, "required")
	return v.Error()
}

// Service stores secrets, encrypts their values, and gives each value only to a rack
// that may read it.
type Service struct {
	aead    cipher.AEAD
	table   *gorp.Table[Key, Secret]
	state   *actions.State[Key, Action]
	indexes indexes
	closer  xio.MultiCloser
	cfg     ServiceConfig
}

// OpenService opens a Service with the given configuration. It returns an error when
// the encryption key cannot decrypt the secrets an earlier start stored. If the error
// is nil, the Service must be closed with Close after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	s = &Service{indexes: newIndexes()}
	if s.cfg, err = config.New(DefaultServiceConfig, cfgs...); err != nil {
		return nil, err
	}
	s.state = actions.NewState[Key, Action](s.cfg.DB)
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.aead, err = openCipher(ctx, s.cfg.DB, s.cfg.EncryptionKey); !ok(err, nil) {
		return nil, err
	}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Secret]{
		DB:              s.cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		Instrumentation: s.cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if s.cfg.Signals == nil {
		return s, nil
	}
	if err = s.cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, Secret]{
		Changes:  s.table.Changes,
		Resource: ResourceID,
	}); !ok(err, nil) {
		return nil, err
	}
	if err = s.cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
		Name:     "sy_secret_action_set",
		Actions:  s.state,
		Resource: ResourceID,
	}); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

// Close releases the resources of the Service. Close is not safe to call concurrently
// with any other method of the Service.
func (s *Service) Close() error { return s.closer.Close() }

// Changes returns the changes that the commit c made to secrets.
func (s *Service) Changes(
	ctx context.Context,
	c gorp.Commit,
) gorp.TxReader[Key, Secret] {
	return s.table.Changes(ctx, c)
}

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// NewWriter opens a Writer that creates, changes, and deletes secrets in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		aead:       s.aead,
		now:        s.cfg.Now,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a query that retrieves secrets. The query never returns a value in
// plain text.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}
