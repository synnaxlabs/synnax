// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import (
	"context"
	"io"
	"iter"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

// ParentRetriever is an interface for retrieving the parent range key for a
// given range. This allows the alias service to implement inheritance without
// a direct dependency on the ranger service.
type ParentRetriever interface {
	RetrieveParentKey(ctx context.Context, key uuid.UUID, tx gorp.Tx) (uuid.UUID, error)
}

// ServiceConfig is the configuration for opening the alias.Service.
type ServiceConfig struct {
	DB              *gorp.DB
	Ontology        *ontology.Ontology
	Search          *search.Index
	Channel         *channel.Service
	Signals         *signals.Provider
	ParentRetriever ParentRetriever
	alamos.Instrumentation
}

var (
	_             config.Config[ServiceConfig] = ServiceConfig{}
	DefaultConfig                              = ServiceConfig{}
)

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("service.ranger.alias")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "search", c.Search)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "parent_retriever", c.ParentRetriever)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Search = override.Nil(c.Search, other.Search)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ParentRetriever = override.Nil(c.ParentRetriever, other.ParentRetriever)
	return c
}

// Service is the main entry point for managing channel aliases on ranges.
type Service struct {
	closer xio.MultiCloser
	table  *gorp.Table[string, Alias]
	cfg    ServiceConfig
}

// OpenService opens a new alias.Service with the provided configuration.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[string, Alias]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if cfg.Signals != nil {
		signalsCfg := signals.GorpPublisherConfigString[Alias](s.table.Observe())
		signalsCfg.SetName = "sy_range_alias_set"
		signalsCfg.DeleteName = "sy_range_alias_delete"
		var sig io.Closer
		if sig, err = signals.PublishFromGorp(ctx, cfg.Signals, signalsCfg); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the service and releases any resources.
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a new Writer to create and delete aliases.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otg:       s.cfg.Ontology,
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		channel:   s.cfg.Channel,
		table:     s.table,
	}
}

// NewReader opens a new Reader to retrieve aliases.
func (s *Service) NewReader(tx gorp.Tx) Reader {
	return Reader{
		tx:              gorp.OverrideTx(s.cfg.DB, tx),
		search:          s.cfg.Search,
		parentRetriever: s.cfg.ParentRetriever,
		table:           s.table,
	}
}

// ontology.Service implementation

var (
	_ ontology.Service = (*Service)(nil)
	_ search.Service   = (*Service)(nil)
)

type change = xchange.Change[string, Alias]

// Type implements ontology.Service.
func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeRangeAlias }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	rangeKey, channelKey, err := parseGorpKey(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var res Alias
	if err = s.table.NewRetrieve().
		Where(gorp.MatchKeys[string, Alias](Alias{Range: rangeKey, Channel: channelKey}.GorpKey())).
		Entry(&res).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(res), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Value.Range, c.Value.Channel).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[string, Alias]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
