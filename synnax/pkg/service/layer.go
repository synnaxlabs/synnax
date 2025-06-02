// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service

import (
	"context"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/layer"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/hardware"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	alamos.Instrumentation
	Distribution *distribution.Layer
	Security     security.Provider
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Distribution = override.Nil(c.Distribution, other.Distribution)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Security = override.Nil(c.Security, other.Security)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("service")
	validate.NotNil(v, "dist", c.Distribution)
	validate.NotNil(v, "security", c.Security)
	return v.Error()

}

type Layer struct {
	DB        *gorp.DB
	User      *user.Service
	RBAC      *rbac.Service
	Token     *token.Service
	Auth      auth.Authenticator
	Ranger    *ranger.Service
	Workspace *workspace.Service
	Schematic *schematic.Service
	LinePlot  *lineplot.Service
	Label     *label.Service
	Log       *log.Service
	Table     *table.Service
	Hardware  *hardware.Service
	Framer    *framer.Service
	closer    xio.MultiCloser
}

func Open(ctx context.Context, cfgs ...Config) (*Layer, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	l := &Layer{DB: cfg.Distribution.Storage.Gorpify()}
	cleanup, ok := layer.NewOpener(ctx, &err, &l.closer)
	defer cleanup()
	if l.User, err = user.NewService(ctx, user.Config{
		DB:       l.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
	}); !ok(nil) {
		return nil, err
	}
	if l.RBAC, err = rbac.NewService(rbac.Config{DB: l.DB}); !ok(nil) {
		return nil, err
	}
	l.Auth = &auth.KV{DB: l.DB}
	if l.Token, err = token.NewService(token.ServiceConfig{
		KeyProvider:      cfg.Security,
		Expiration:       24 * time.Hour,
		RefreshThreshold: 1 * time.Hour,
	}); !ok(nil) {
		return nil, err
	}
	if l.Ranger, err = ranger.OpenService(ctx, ranger.Config{
		DB:       l.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
		Signals:  cfg.Distribution.Signals,
	}); !ok(l.Ranger) {
		return nil, err
	}
	if l.Workspace, err = workspace.NewService(ctx, workspace.Config{
		DB:       l.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
	}); !ok(nil) {
		return nil, err
	}
	if l.Schematic, err = schematic.NewService(ctx, schematic.Config{
		DB:       l.DB,
		Ontology: cfg.Distribution.Ontology,
	}); !ok(nil) {
		return nil, err
	}
	if l.LinePlot, err = lineplot.NewService(ctx, lineplot.Config{
		DB:       l.DB,
		Ontology: cfg.Distribution.Ontology,
	}); !ok(nil) {
		return nil, err
	}
	if l.Log, err = log.NewService(ctx, log.Config{
		DB:       l.DB,
		Ontology: cfg.Distribution.Ontology,
	}); !ok(nil) {
		return nil, err
	}
	if l.Table, err = table.NewService(ctx, table.Config{
		DB:       l.DB,
		Ontology: cfg.Distribution.Ontology,
	}); !ok(nil) {
		return nil, err
	}
	if l.Label, err = label.OpenService(ctx, label.Config{
		DB:       l.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
		Signals:  cfg.Distribution.Signals,
	}); !ok(l.Label) {
		return nil, err
	}
	if l.Hardware, err = hardware.OpenService(ctx, hardware.Config{
		Instrumentation: cfg.Instrumentation.Child("hardware"),
		DB:              l.DB,
		Ontology:        cfg.Distribution.Ontology,
		Group:           cfg.Distribution.Group,
		HostProvider:    cfg.Distribution.Cluster,
		Signals:         cfg.Distribution.Signals,
		Channel:         cfg.Distribution.Channel,
		Framer:          cfg.Distribution.Framer,
	}); !ok(l.Hardware) {
		return nil, err
	}
	if l.Framer, err = framer.OpenService(
		ctx,
		framer.Config{
			Instrumentation: cfg.Instrumentation.Child("framer"),
			Framer:          cfg.Distribution.Framer,
			Channel:         cfg.Distribution.Channel,
		},
	); !ok(l.Framer) {
		return nil, err
	}
	return l, nil
}

func (l *Layer) Close() error { return l.closer.Close() }
