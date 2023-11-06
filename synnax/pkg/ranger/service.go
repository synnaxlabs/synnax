// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/cdc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"io"
	"sync"
)

type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Group    *group.Service
	CDC      *cdc.Provider
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("ranger")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	validate.NotNil(v, "Group", c.Group)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.CDC = override.Nil(c.CDC, other.CDC)
	return c
}

type Service struct {
	Config
	group                 group.Group
	cdc                   io.Closer
	mu                    sync.Mutex
	activeRange           uuid.UUID
	activeRangeObservable observe.Observer[[]changex.Change[[]byte, struct{}]]
}

const groupName = "Ranges"

func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	cfg.Ontology.RegisterService(&aliasOntologyService{db: cfg.DB})
	if cfg.CDC != nil {
		rangeCDC, err := cdc.SubscribeToGorp(ctx, cfg.CDC, cdc.GorpConfigUUID[Range](cfg.DB))
		if err != nil {
			return nil, err
		}
		aliasCDCCfg := cdc.GorpConfigString[alias](cfg.DB)
		aliasCDCCfg.SetName = "sy_range_alias_set"
		aliasCDCCfg.DeleteName = "sy_range_alias_delete"
		aliasCDC, err := cdc.SubscribeToGorp(ctx, cfg.CDC, aliasCDCCfg)
		if err != nil {
			return nil, err
		}
		s.activeRangeObservable = observe.New[[]changex.Change[[]byte, struct{}]]()
		activeRangeCDC, err := cfg.CDC.SubscribeToObservable(ctx, cdc.ObservableConfig{
			Name:       "sy_active_range",
			Set:        channel.Channel{Name: "sy_active_range_set", DataType: telem.UUIDT},
			Delete:     channel.Channel{Name: "sy_active_range_clear", DataType: telem.UUIDT},
			Observable: s.activeRangeObservable,
		})
		if err != nil {
			return nil, err
		}
		s.cdc = xio.MultiCloser{rangeCDC, aliasCDC, activeRangeCDC}
	}
	return s, err
}

func (s *Service) Close() error {
	if s.cdc != nil {
		return s.cdc.Close()
	}
	return nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    tx,
		otg:   s.Ontology.NewWriter(tx),
		group: s.group,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return newRetrieve(s.DB, s.Ontology)
}

func (s *Service) SetActiveRange(ctx context.Context, key uuid.UUID, tx gorp.Tx) error {
	if err := s.NewRetrieve().WhereKeys(key).Exec(ctx, tx); err != nil {
		return err
	}
	s.mu.Lock()
	s.activeRange = key
	s.activeRangeObservable.Notify(ctx, []changex.Change[[]byte, struct{}]{{
		Variant: changex.Set,
		Key:     key[:],
	}})
	s.mu.Unlock()
	return nil
}

func (s *Service) RetrieveActiveRange(ctx context.Context, tx gorp.Tx) (r Range, err error) {
	s.mu.Lock()
	err = s.NewRetrieve().WhereKeys(s.activeRange).Entry(&r).Exec(ctx, tx)
	s.mu.Unlock()
	return r, err
}

func (s *Service) ClearActiveRange(ctx context.Context) {
	s.mu.Lock()
	key := s.activeRange
	s.activeRangeObservable.Notify(ctx, []changex.Change[[]byte, struct{}]{{
		Variant: changex.Delete,
		Key:     key[:],
	}})
	s.activeRange = uuid.Nil
	s.mu.Unlock()
}
