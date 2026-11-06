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
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
	// Ontology holds a resource for each secret.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group holds the "Secrets" group, the default parent of every secret.
	//
	// [REQUIRED]
	Group *group.Service
	// Signals publishes changes to secrets.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// Channel finds the channels that Signals publishes changes on.
	//
	// [REQUIRED] when Signals is set.
	Channel *channel.Service
	// Uses finds the resources that use a secret, for delete and for the rack rule of
	// Resolve.
	//
	// [REQUIRED]
	Uses Finder
	// Grant stores the grants that let a rack read secrets that nothing on it uses.
	//
	// [REQUIRED]
	Grant *grant.Service
	// Now returns the current time.
	//
	// [OPTIONAL] - Defaults to telem.Now.
	Now func() telem.TimeStamp
	alamos.Instrumentation
	// EncryptionKey protects the value of every stored secret. It encrypts the random
	// data key that encrypts each value.
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
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Uses = override.Nil(c.Uses, other.Uses)
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
	v.NotNil("ontology", c.Ontology)
	v.NotNil("group", c.Group)
	v.NotNil("uses", c.Uses)
	v.NotNil("grant", c.Grant)
	v.NotNil("now", c.Now)
	v.Ternary(
		"channel",
		c.Signals != nil && c.Channel == nil,
		"required when signals is set",
	)
	v.Ternary("encryption_key", c.EncryptionKey == EncryptionKey{}, "required")
	return v.Error()
}

// Service stores secrets, encrypts their values, and gives each value only to a rack
// that may read it.
type Service struct {
	aead             cipher.AEAD
	group            group.Group
	table            *gorp.Table[Key, Secret]
	indexes          indexes
	closer           xio.MultiCloser
	cfg              ServiceConfig
	setChannelKey    channel.Key
	deleteChannelKey channel.Key
}

// OpenService opens a Service with the given configuration. It returns an error when
// the encryption key cannot decrypt the secrets an earlier start stored. If the error
// is nil, the Service must be closed with Close after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	s = &Service{indexes: newIndexes()}
	if s.cfg, err = config.New(DefaultServiceConfig, cfgs...); err != nil {
		return nil, err
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.aead, err = openCipher(ctx, s.cfg.DB, s.cfg.EncryptionKey); !ok(err, nil) {
		return nil, err
	}
	if s.group, err = s.cfg.Group.CreateOrRetrieve(
		ctx,
		"Secrets",
		ontology.RootID,
	); !ok(err, nil) {
		return nil, err
	}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Secret]{
		DB:              s.cfg.DB,
		Indexes:         s.indexes.all(),
		Instrumentation: s.cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	s.cfg.Ontology.RegisterService(s)
	if s.cfg.Signals == nil {
		return s, nil
	}
	var sig io.Closer
	if sig, err = s.cfg.Signals.PublishFromGorp(
		ctx,
		signals.GorpPublisherConfigUUID(s.table.Observe()),
	); !ok(err, sig) {
		return nil, err
	}
	if s.setChannelKey, err = s.retrieveChannelKey(
		ctx,
		"sy_secret_set",
	); !ok(err, nil) {
		return nil, err
	}
	if s.deleteChannelKey, err = s.retrieveChannelKey(
		ctx,
		"sy_secret_delete",
	); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

func (s *Service) retrieveChannelKey(
	ctx context.Context,
	name string,
) (channel.Key, error) {
	var ch channel.Channel
	if err := s.cfg.Channel.NewRetrieve().
		Where(channel.MatchNames(name)).
		Entry(&ch).
		Exec(ctx, nil); err != nil {
		return 0, err
	}
	return ch.Key(), nil
}

// SetChannelKey returns the key of the channel that publishes each secret that is
// created or changed. Its samples carry the names and paths of secrets, so only
// subjects that can retrieve a secret may receive them. The key is zero when the
// Service has no Signals.
func (s *Service) SetChannelKey() channel.Key { return s.setChannelKey }

// DeleteChannelKey returns the key of the channel that publishes the key of each
// deleted secret. Only subjects that can retrieve a secret may receive its samples. The
// key is zero when the Service has no Signals.
func (s *Service) DeleteChannelKey() channel.Key { return s.deleteChannelKey }

// Close releases the resources of the Service. Close is not safe to call concurrently
// with any other method of the Service.
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a Writer that creates, renames, and deletes secrets. If tx is nil,
// the Writer writes directly to the DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		group:     s.group,
		table:     s.table,
		aead:      s.aead,
		uses:      s.cfg.Uses,
		now:       s.cfg.Now,
	}
}

// NewRetrieve opens a query that retrieves secrets. The query never returns a value in
// plain text.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		baseTX:  s.cfg.DB,
		indexes: s.indexes,
	}
}
