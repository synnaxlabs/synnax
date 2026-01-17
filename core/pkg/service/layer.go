// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/console"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening the service layer. See fields for
// details on defining the configuration.
type Config struct {
	// Instrumentation is for logging, tracing, metrics, etc.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Distribution is the underlying distribution layer.
	//
	// [REQUIRED]
	Distribution *distribution.Layer
	// Security provides TLS certificates and encryption keys for the service layer.
	//
	// [REQUIRED]
	Security security.Provider
	// Storage is the storage layer used for disk usage metrics.
	//
	// [REQUIRED]
	Storage *storage.Layer
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening the service layer.
	// This configuration is not valid on its own and must be overridden by the
	// required fields specified in Config.
	DefaultConfig = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Distribution = override.Nil(c.Distribution, other.Distribution)
	c.Security = override.Nil(c.Security, other.Security)
	c.Storage = override.Nil(c.Storage, other.Storage)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("service")
	validate.NotNil(v, "distribution", c.Distribution)
	validate.NotNil(v, "security", c.Security)
	return v.Error()

}

// Layer contains all relevant services within the Synnax service layer.
// The service layer wraps the distribution layer to provide the core services of
// synnax that do not require network awareness.
type Layer struct {
	// User is the service for registering and retrieving information about users.
	User *user.Service
	// RBAC implements role-based access control for users.
	RBAC *rbac.Service
	// Token is for creating and validating authentication tokens.
	Token *token.Service
	// Auth is for authenticating users with credentials.
	Auth auth.Authenticator
	// Ranger is for working with ranges.
	Ranger *ranger.Service
	// Alias is for working with channel aliases on ranges.
	Alias *alias.Service
	// KV is for working with key-value pairs on ranges.
	KV *kv.Service
	// Workspace is for working with Workspaces.
	Workspace *workspace.Service
	// Schematic is for working with schematic visualizations.
	Schematic *schematic.Service
	// LinePlot is for working with line plot visualizations.
	LinePlot *lineplot.Service
	// Log is for working with log visualizations.
	Log *log.Service
	// Table is for working with table visualizations.
	Table *table.Service
	// Label is for working with user-defined labels that can be attached to various
	// data structures within Synnax.
	Label  *label.Service
	Rack   *rack.Service
	Task   *task.Service
	Device *device.Service
	// Framer is for reading, writing, and streaming frames of telemetry from channels
	// across the cluster.
	Framer *framer.Service
	// Console is for serving the web-based console UI.
	Console *console.Service
	// Arc is used for validating, saving, and executing arc automations.
	Arc *arc.Service
	// Metrics is used for collecting host machine metrics and publishing them over channels
	Metrics *metrics.Service
	// Status is used for tracking the statuses
	Status *status.Service
	// View is used for managing views
	View *view.Service
	// closer is for properly shutting down the service layer.
	closer xio.MultiCloser
}

// Close shuts down the service layer, returning any error encountered.
func (l *Layer) Close() error { return l.closer.Close() }

// Open opens the service layer using the provided configurations. Later configurations
// override the fields set in previous ones. If the configuration is invalid, or
// any services fail to open, Open returns a nil layer and an error. If the returned
// error is nil, the Layer must be closed by calling Close after use.
func Open(ctx context.Context, cfgs ...Config) (*Layer, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	l := &Layer{}
	cleanup, ok := service.NewOpener(ctx, &l.closer)
	defer func() {
		err = cleanup(err)
	}()

	if l.User, err = user.OpenService(ctx, user.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.RBAC, err = rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
		Signals:  cfg.Distribution.Signals,
		Group:    cfg.Distribution.Group,
	}); !ok(err, nil) {
		return nil, err
	}

	l.Auth = &auth.KV{DB: cfg.Distribution.DB}
	if l.Token, err = token.NewService(token.ServiceConfig{
		KeyProvider:      cfg.Security,
		Expiration:       24 * time.Hour,
		RefreshThreshold: 1 * time.Hour,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Label, err = label.OpenService(ctx, label.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
		Signals:  cfg.Distribution.Signals,
	}); !ok(err, l.Label) {
		return nil, err
	}
	if l.Ranger, err = ranger.OpenService(ctx, ranger.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
		Signals:  cfg.Distribution.Signals,
		Label:    l.Label,
	}); !ok(err, l.Ranger) {
		return nil, err
	}
	if l.Alias, err = alias.OpenService(ctx, alias.ServiceConfig{
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Signals:         cfg.Distribution.Signals,
		ParentRetriever: l.Ranger,
	}); !ok(err, l.Alias) {
		return nil, err
	}
	if l.KV, err = kv.OpenService(ctx, kv.ServiceConfig{
		DB:      cfg.Distribution.DB,
		Signals: cfg.Distribution.Signals,
	}); !ok(err, l.KV) {
		return nil, err
	}
	if l.Workspace, err = workspace.OpenService(ctx, workspace.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
		Signals:  cfg.Distribution.Signals,
	}); !ok(err, l.Workspace) {
		return nil, err
	}
	if l.Schematic, err = schematic.OpenService(ctx, schematic.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
		Signals:  cfg.Distribution.Signals,
	}); !ok(err, l.Schematic) {
		return nil, err
	}
	if l.LinePlot, err = lineplot.NewService(lineplot.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Log, err = log.NewService(log.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Table, err = table.NewService(table.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Status, err = status.OpenService(
		ctx,
		status.ServiceConfig{
			Instrumentation: cfg.Child("status"),
			DB:              cfg.Distribution.DB,
			Signals:         cfg.Distribution.Signals,
			Ontology:        cfg.Distribution.Ontology,
			Group:           cfg.Distribution.Group,
			Label:           l.Label,
		},
	); !ok(err, l.Status) {
		return nil, err
	}
	if l.Rack, err = rack.OpenService(ctx, rack.ServiceConfig{
		Instrumentation: cfg.Child("rack"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Group:           cfg.Distribution.Group,
		HostProvider:    cfg.Distribution.Cluster,
		Signals:         cfg.Distribution.Signals,
		Status:          l.Status,
	}); !ok(err, l.Rack) {
		return nil, err
	}
	if l.Device, err = device.OpenService(ctx, device.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Ontology: cfg.Distribution.Ontology,
		Group:    cfg.Distribution.Group,
		Signals:  cfg.Distribution.Signals,
		Status:   l.Status,
		Rack:     l.Rack,
	}); !ok(err, l.Device) {
		return nil, err
	}
	if l.Task, err = task.OpenService(ctx, task.ServiceConfig{
		Instrumentation: cfg.Child("task"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Group:           cfg.Distribution.Group,
		Signals:         cfg.Distribution.Signals,
		Channel:         cfg.Distribution.Channel,
		Rack:            l.Rack,
		Status:          l.Status,
	}); !ok(err, l.Task) {
		return nil, err
	}
	if l.Arc, err = arc.OpenService(
		ctx,
		arc.ServiceConfig{
			Instrumentation: cfg.Child("arc"),
			DB:              cfg.Distribution.DB,
			Ontology:        cfg.Distribution.Ontology,
			Framer:          cfg.Distribution.Framer,
			Channel:         cfg.Distribution.Channel,
			Signals:         cfg.Distribution.Signals,
			Status:          l.Status,
		},
	); !ok(err, l.Arc) {
		return nil, err
	}
	if l.View, err = view.OpenService(
		ctx,
		view.ServiceConfig{
			Instrumentation: cfg.Child("view"),
			DB:              cfg.Distribution.DB,
			Signals:         cfg.Distribution.Signals,
			Ontology:        cfg.Distribution.Ontology,
			Group:           cfg.Distribution.Group,
		},
	); !ok(err, l.View) {
		return nil, err
	}
	cfg.Distribution.Channel.SetCalculationAnalyzer(l.Arc.AnalyzeCalculation)
	if l.Framer, err = framer.OpenService(
		ctx,
		framer.ServiceConfig{
			DB:              cfg.Distribution.DB,
			Instrumentation: cfg.Child("framer"),
			Framer:          cfg.Distribution.Framer,
			Channel:         cfg.Distribution.Channel,
			Arc:             l.Arc,
		},
	); !ok(err, l.Framer) {
		return nil, err
	}
	l.Console = console.NewService()
	if l.Metrics, err = metrics.OpenService(
		ctx,
		metrics.ServiceConfig{
			Instrumentation: cfg.Child("metrics"),
			Framer:          l.Framer,
			Channel:         cfg.Distribution.Channel,
			HostProvider:    cfg.Distribution.Cluster,
			Storage:         cfg.Storage,
		}); !ok(err, l.Metrics) {
		return nil, err
	}
	return l, nil
}
