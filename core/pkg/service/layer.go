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
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	channelsignals "github.com/synnaxlabs/synnax/pkg/service/channel/signals"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	ontologysignals "github.com/synnaxlabs/synnax/pkg/service/ontology/signals"
	pdruntime "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// LayerConfig is the configuration for opening the service layer. See fields for
// details on defining the configuration.
type LayerConfig struct {
	// Security provides TLS certificates and encryption keys for the service layer.
	//
	// [REQUIRED]
	Security security.Provider
	// Distribution is the underlying distribution layer.
	//
	// [REQUIRED]
	Distribution *distribution.Layer
	// Storage is the storage layer used for disk usage metrics.
	//
	// [REQUIRED]
	Storage *storage.Layer
	// RootCredentials are the credentials for the root user. If provided, the root
	// user will be provisioned during service layer initialization.
	//
	// [OPTIONAL]
	RootCredentials auth.Credentials
	// Instrumentation is for logging, tracing, metrics, etc.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var (
	_ config.Config[LayerConfig] = LayerConfig{}
	// DefaultLayerConfig is the default configuration for opening the service layer.
	// This configuration is not valid on its own and must be overridden by the
	// required fields specified in Config.
	DefaultLayerConfig = LayerConfig{}
)

// Override implements config.Config.
func (c LayerConfig) Override(other LayerConfig) LayerConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Distribution = override.Nil(c.Distribution, other.Distribution)
	c.Security = override.Nil(c.Security, other.Security)
	c.Storage = override.Nil(c.Storage, other.Storage)
	c.RootCredentials = override.Zero(c.RootCredentials, other.RootCredentials)
	return c
}

// Validate implements config.Config.
func (c LayerConfig) Validate() error {
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
	// Auth validates credentials and manages credential storage.
	Auth *auth.Service
	// Token is for creating and validating authentication tokens.
	Token *token.Service
	// Ranger is for working with ranges.
	Ranger *ranger.Service
	// Alias is for working with channel aliases on ranges.
	Alias *alias.Service
	// KV is for working with key-value pairs on ranges.
	KV *kv.Service
	// Project is for working with Projects.
	Project *project.Service
	// Schematic is for working with schematic visualizations.
	Schematic *schematic.Service
	// LinePlot is for working with line plot visualizations.
	LinePlot *lineplot.Service
	// Log is for working with log visualizations.
	Log *log.Service
	// Table is for working with table visualizations.
	Table *table.Service
	// Panel is for working with Panels.
	Panel *panel.Service
	// Label is for working with user-defined labels that can be attached to various
	// data structures within Synnax.
	Label  *label.Service
	Rack   *rack.Service
	Task   *task.Service
	Device *device.Service
	// Framer is for reading, writing, and streaming frames of telemetry from channels
	// across the cluster.
	Framer *framer.Service
	// Channel is the highest-level channel service and owns calculated channel behavior.
	Channel *channel.Service
	// Arc is used for validating, saving, and executing arc automations.
	Arc *arc.Service
	// Metrics is used for collecting host machine metrics and publishing them over channels
	Metrics *metrics.Service
	// Status is used for tracking the statuses
	Status *status.Service
	// View is used for managing views
	View *view.Service
	// ImEx is the central import/export registry.
	ImEx *imex.Service
	// Node publishes the cluster's nodes as resources in the ontology and search
	// index.
	Node *node.Service
	// Driver is the Go task executor that handles in-process task lifecycle.
	Driver *driver.Driver
	// Signals propagates changes to distribution and service layer data structures
	// through free channels in Synnax.
	Signals *signals.Provider
	// closer is for properly shutting down the service layer.
	closer io.MultiCloser
}

// Close shuts down the service layer, returning any error encountered.
func (l *Layer) Close() error { return l.closer.Close() }

// OpenLayer opens the service layer using the provided configurations. Later configurations
// override the fields set in previous ones. If the configuration is invalid, or
// any services fail to open, Open returns a nil layer and an error. If the returned
// error is nil, the Layer must be closed by calling Close after use.
func OpenLayer(ctx context.Context, cfgs ...LayerConfig) (l *Layer, err error) {
	cfg, err := config.New(DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	l = &Layer{}
	cleanup, ok := service.NewOpener(ctx, &l.closer)
	defer func() {
		err = cleanup(err)
	}()
	if l.Node, err = node.NewService(ctx, node.ServiceConfig{
		Instrumentation: cfg.Child("node"),
		Cluster:         cfg.Distribution.Cluster,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Auth, err = auth.OpenService(ctx, auth.ServiceConfig{
		Instrumentation: cfg.Child("auth"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.Auth) {
		return nil, err
	}
	if l.Label, err = label.OpenService(ctx, label.ServiceConfig{
		Instrumentation: cfg.Child("label"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Group:           cfg.Distribution.Group,
	}); !ok(err, l.Label) {
		return nil, err
	}
	if l.Status, err = status.OpenService(
		ctx,
		status.ServiceConfig{
			Instrumentation: cfg.Child("status"),
			DB:              cfg.Distribution.DB,
			Ontology:        cfg.Distribution.Ontology,
			Search:          cfg.Distribution.Search,
			Group:           cfg.Distribution.Group,
			Label:           l.Label,
		},
	); !ok(err, l.Status) {
		return nil, err
	}
	if l.Channel, err = channel.NewService(ctx, channel.ServiceConfig{
		Instrumentation: cfg.Child("channel"),
		Channel:         cfg.Distribution.Channel,
		Status:          l.Status,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Framer, err = framer.OpenService(
		ctx,
		framer.ServiceConfig{
			Instrumentation: cfg.Child("framer"),
			Framer:          cfg.Distribution.Framer,
			Channel:         l.Channel,
			Status:          l.Status,
		},
	); !ok(err, l.Framer) {
		return nil, err
	}
	if l.Signals, err = signals.New(signals.Config{
		Channel:         l.Channel,
		Framer:          l.Framer,
		Instrumentation: cfg.Child("signals"),
	}); !ok(err, nil) {
		return nil, err
	}
	if closer, err := channelsignals.Publish(
		ctx,
		l.Signals,
		l.Channel.Observe(),
	); !ok(err, closer) {
		return nil, err
	}
	if closer, err := signals.PublishFromGorp(
		ctx,
		l.Signals,
		signals.GorpPublisherConfigUUID(cfg.Distribution.Group.Observe()),
	); !ok(err, closer) {
		return nil, err
	}
	if closer, err := ontologysignals.Publish(
		ctx,
		l.Signals,
		cfg.Distribution.Ontology,
	); !ok(err, closer) {
		return nil, err
	}
	if closer, err := signals.PublishFromGorp(
		ctx,
		l.Signals,
		signals.GorpPublisherConfigUUID(l.Label.Observe()),
	); !ok(err, closer) {
		return nil, err
	}
	if closer, err := signals.PublishFromGorp(
		ctx,
		l.Signals,
		signals.GorpPublisherConfigString(l.Status.Observe()),
	); !ok(err, closer) {
		return nil, err
	}
	if l.User, err = user.OpenService(ctx, user.ServiceConfig{
		Instrumentation: cfg.Child("user"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Group:           cfg.Distribution.Group,
		Signals:         l.Signals,
		Auth:            l.Auth,
		RootCredentials: cfg.RootCredentials,
	}); !ok(err, l.User) {
		return nil, err
	}
	if l.RBAC, err = rbac.OpenService(ctx, rbac.ServiceConfig{
		Instrumentation: cfg.Child("rbac"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Signals:         l.Signals,
		Group:           cfg.Distribution.Group,
		Search:          cfg.Distribution.Search,
		User:            l.User,
	}); !ok(err, l.RBAC) {
		return nil, err
	}
	if l.Token, err = token.NewService(token.ServiceConfig{
		KeyProvider:      cfg.Security,
		Expiration:       24 * time.Hour,
		RefreshThreshold: 1 * time.Hour,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Ranger, err = ranger.OpenService(ctx, ranger.ServiceConfig{
		Instrumentation: cfg.Child("ranger"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Group:           cfg.Distribution.Group,
		Signals:         l.Signals,
		Label:           l.Label,
	}); !ok(err, l.Ranger) {
		return nil, err
	}
	if l.KV, err = kv.OpenService(ctx, kv.ServiceConfig{
		Instrumentation: cfg.Child("kv"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
	}); !ok(err, l.KV) {
		return nil, err
	}
	if l.Project, err = project.OpenService(ctx, project.ServiceConfig{
		Instrumentation: cfg.Child("project"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Group:           cfg.Distribution.Group,
		Signals:         l.Signals,
	}); !ok(err, l.Project) {
		return nil, err
	}
	if l.Schematic, err = schematic.OpenService(ctx, schematic.ServiceConfig{
		Instrumentation: cfg.Child("schematic"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Group:           cfg.Distribution.Group,
		Signals:         l.Signals,
	}); !ok(err, l.Schematic) {
		return nil, err
	}
	if l.LinePlot, err = lineplot.OpenService(ctx, lineplot.ServiceConfig{
		Instrumentation: cfg.Child("lineplot"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Signals:         l.Signals,
	}); !ok(err, l.LinePlot) {
		return nil, err
	}
	l.ImEx = imex.NewService(cfg.Distribution.Ontology)
	if l.Log, err = log.OpenService(ctx, log.ServiceConfig{
		Instrumentation: cfg.Child("log"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Signals:         l.Signals,
		ImEx:            l.ImEx,
	}); !ok(err, l.Log) {
		return nil, err
	}
	if l.Panel, err = panel.OpenService(ctx, panel.ServiceConfig{
		Instrumentation: cfg.Child("panel"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Signals:         l.Signals,
	}); !ok(err, l.Panel) {
		return nil, err
	}
	if l.Table, err = table.OpenService(ctx, table.ServiceConfig{
		Instrumentation: cfg.Child("table"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Signals:         l.Signals,
	}); !ok(err, l.Table) {
		return nil, err
	}
	if l.Rack, err = rack.OpenService(ctx, rack.ServiceConfig{
		Instrumentation: cfg.Child("rack"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Group:           cfg.Distribution.Group,
		HostProvider:    cfg.Distribution.Cluster,
		Status:          l.Status,
	}); !ok(err, l.Rack) {
		return nil, err
	}
	if closer, err := signals.PublishFromGorp(
		ctx,
		l.Signals,
		signals.GorpPublisherConfigNumeric(l.Rack.Observe(), telem.Uint32T),
	); !ok(err, closer) {
		return nil, err
	}
	if l.Device, err = device.OpenService(ctx, device.ServiceConfig{
		Instrumentation: cfg.Child("device"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Group:           cfg.Distribution.Group,
		Signals:         l.Signals,
		Status:          l.Status,
		Rack:            l.Rack,
	}); !ok(err, l.Device) {
		return nil, err
	}
	if l.Task, err = task.OpenService(ctx, task.ServiceConfig{
		Instrumentation: cfg.Child("task"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Group:           cfg.Distribution.Group,
		Channel:         l.Channel,
		Rack:            l.Rack,
		Status:          l.Status,
	}); !ok(err, l.Task) {
		return nil, err
	}
	if closer, err := signals.PublishFromGorp(
		ctx,
		l.Signals,
		signals.GorpPublisherConfigPureNumeric(l.Task.Observe(), telem.Uint64T),
	); !ok(err, closer) {
		return nil, err
	}
	if l.Arc, err = arc.OpenService(
		ctx,
		arc.ServiceConfig{
			Instrumentation: cfg.Child("arc"),
			DB:              cfg.Distribution.DB,
			Ontology:        cfg.Distribution.Ontology,
			Search:          cfg.Distribution.Search,
			Channel:         l.Channel,
			Task:            l.Task,
			Signals:         l.Signals,
		},
	); !ok(err, l.Arc) {
		return nil, err
	}
	if closer, err := calcgraph.Open(ctx, calcgraph.Config{
		Instrumentation: cfg.Child("channel.calculation.graph"),
		Channel:         l.Channel,
		Status:          l.Status,
	}); !ok(err, closer) {
		return nil, err
	}
	if l.Alias, err = alias.OpenService(ctx, alias.ServiceConfig{
		Instrumentation: cfg.Child("alias"),
		DB:              cfg.Distribution.DB,
		Ontology:        cfg.Distribution.Ontology,
		Search:          cfg.Distribution.Search,
		Signals:         l.Signals,
		Channel:         l.Channel,
		ParentRetriever: l.Ranger,
	}); !ok(err, l.Alias) {
		return nil, err
	}
	if l.View, err = view.OpenService(
		ctx,
		view.ServiceConfig{
			Instrumentation: cfg.Child("view"),
			DB:              cfg.Distribution.DB,
			Signals:         l.Signals,
			Ontology:        cfg.Distribution.Ontology,
			Search:          cfg.Distribution.Search,
			Group:           cfg.Distribution.Group,
		},
	); !ok(err, l.View) {
		return nil, err
	}
	if l.Metrics, err = metrics.OpenService(
		ctx,
		metrics.ServiceConfig{
			Instrumentation: cfg.Child("metrics"),
			DB:              cfg.Distribution.DB,
			Framer:          l.Framer,
			Channel:         l.Channel,
			HostProvider:    cfg.Distribution.Cluster,
			Storage:         cfg.Storage,
			Group:           cfg.Distribution.Group,
			Ontology:        cfg.Distribution.Ontology,
		}); !ok(err, l.Metrics) {
		return nil, err
	}
	arcFactory, err := arctask.NewFactory(arctask.FactoryConfig{
		Instrumentation: cfg.Child("arc.task"),
		Channel:         l.Channel,
		Framer:          l.Framer,
		Status:          l.Status,
		GetProgram:      l.Arc.CompileProgram,
		Ranger:          l.Ranger,
	})
	if !ok(err, nil) {
		return nil, err
	}
	pdFactory, err := pdruntime.NewFactory(pdruntime.FactoryConfig{
		Instrumentation: cfg.Child("pagerduty"),
		Status:          l.Status,
	})
	if !ok(err, nil) {
		return nil, err
	}
	if l.Driver, err = driver.Open(ctx, driver.Config{
		Instrumentation: cfg.Child("driver"),
		DB:              cfg.Distribution.DB,
		Rack:            l.Rack,
		Task:            l.Task,
		Framer:          l.Framer,
		Channel:         l.Channel,
		Status:          l.Status,
		Factories:       []driver.Factory{arcFactory, pdFactory},
		Host:            cfg.Distribution.Cluster,
	}); !ok(err, l.Driver) {
		return nil, err
	}
	return l, nil
}
