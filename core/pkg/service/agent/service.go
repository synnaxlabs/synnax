// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package agent

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/agent/llm"
	sarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type ServiceConfig struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Channel  *channel.Service
	Arc      *sarc.Service
	Task     *task.Service
	Framer   *framer.Service
	Signals  *signals.Provider
	LLM      llm.Config
	RackKey  rack.Key
	alamos.Instrumentation
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Arc = override.Nil(c.Arc, other.Arc)
	c.Task = override.Nil(c.Task, other.Task)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.LLM = c.LLM.Override(other.LLM)
	c.RackKey = override.Numeric(c.RackKey, other.RackKey)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

func (c ServiceConfig) Validate() error {
	v := validate.New("agent")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "arc", c.Arc)
	validate.NotNil(v, "task", c.Task)
	return v.Error()
}

type Service struct {
	cfg       ServiceConfig
	generator llm.Generator
	providers []ContextProvider
	closer    io.Closer
}

func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if cfg.LLM.APIKey != "" {
		s.generator, err = llm.NewGenerator(cfg.LLM)
		if err != nil {
			return nil, err
		}
	}
	s.providers = []ContextProvider{
		&ChannelContextProvider{Channel: cfg.Channel, Framer: cfg.Framer},
		&HistoryContextProvider{DB: cfg.DB, Arc: cfg.Arc},
	}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals != nil {
		s.closer, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[Agent](cfg.DB),
		)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:              gorp.OverrideTx(s.cfg.DB, tx),
		otg:             s.cfg.Ontology.NewWriter(tx),
		generator:       s.generator,
		arc:             s.cfg.Arc,
		task:            s.cfg.Task,
		Instrumentation: s.cfg.Instrumentation,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, Agent](),
		baseTX: s.cfg.DB,
		otg:    s.cfg.Ontology,
	}
}

// GenerateAndDeploy runs the LLM generation outside of a database transaction,
// then deploys the resulting Arc code within a short-lived transaction.
func (s *Service) GenerateAndDeploy(ctx context.Context, a *Agent) error {
	if s.generator == nil {
		return errors.New("[agent] - LLM API key not configured. Set the SYNNAX_LLM_API_KEY environment variable to enable agent features.")
	}
	a.RackKey = s.cfg.RackKey
	g := generator{
		generator:       s.generator,
		providers:       s.providers,
		arc:             s.cfg.Arc,
		Instrumentation: s.cfg.Instrumentation,
	}
	code, response, err := g.generate(ctx, a)
	if err != nil {
		return s.failAgent(ctx, a, err)
	}
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.NewWriter(tx)
		return w.deploy(ctx, a, code, response)
	})
}

func (s *Service) failAgent(ctx context.Context, a *Agent, err error) error {
	a.State = StateError
	s.cfg.Instrumentation.L.Error("agent failed", zap.Error(err), zap.Stringer("key", a.Key))
	_ = s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return gorp.NewCreate[uuid.UUID, Agent]().Entry(a).Exec(ctx, tx)
	})
	return err
}
