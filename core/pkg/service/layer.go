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
	"slices"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	passwordversions "github.com/synnaxlabs/synnax/pkg/service/auth/password/versions"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	channelsignals "github.com/synnaxlabs/synnax/pkg/service/channel/signals"
	"github.com/synnaxlabs/synnax/pkg/service/control"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/history"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/synnax/pkg/service/modbus"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	pdruntime "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/project/bundle"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktask "github.com/synnaxlabs/synnax/pkg/service/rack/task"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
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
	// FIPS restricts password hashing to PBKDF2, the one password hash NIST approves.
	//
	// [OPTIONAL] - Defaults to false.
	FIPS *bool
	// SessionIdleTimeout ends a session that no request has used for this long.
	//
	// [OPTIONAL] - Defaults to 7 days.
	SessionIdleTimeout time.Duration
	// License configures the license service. The layer supplies its DB and
	// instrumentation.
	//
	// [OPTIONAL] - Defaults to license.DefaultServiceConfig.
	License license.ServiceConfig
	// SecretsKey protects the value of every stored secret. It encrypts the random data
	// key that encrypts each value.
	//
	// [REQUIRED]
	SecretsKey secret.EncryptionKey
	// ValidateChannelNames enables channel name validation during creation and
	// renaming. When false, channels may have names with spaces, special characters,
	// etc.
	//
	// [OPTIONAL] - Defaults to true (validation enabled)
	ValidateChannelNames *bool
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
	c.FIPS = override.Nil(c.FIPS, other.FIPS)
	c.SessionIdleTimeout = override.Numeric(
		c.SessionIdleTimeout,
		other.SessionIdleTimeout,
	)
	c.License = c.License.Override(other.License)
	c.ValidateChannelNames = override.Nil(
		c.ValidateChannelNames, other.ValidateChannelNames,
	)
	c.SecretsKey = override.If(
		c.SecretsKey,
		other.SecretsKey,
		other.SecretsKey != secret.EncryptionKey{},
	)
	return c
}

// Validate implements config.Config.
func (c LayerConfig) Validate() error {
	v := validate.New("service")
	v.NotNil("distribution", c.Distribution)
	v.NotNil("security", c.Security)
	v.Ternary("secrets_key", c.SecretsKey == secret.EncryptionKey{}, "required")
	return v.Error()
}

// Layer contains all relevant services within the Synnax service layer.
// The service layer wraps the distribution layer to provide the core services of
// synnax that do not require network awareness.
type Layer struct {
	// Group is for grouping related resources.
	Group *group.Service
	// User is the service for registering and retrieving information about users.
	User *user.Service
	// RBAC implements role-based access control for users.
	RBAC *rbac.Service
	// Password stores passwords and checks proofs of them.
	Password *password.Service
	// APIKey stores API keys and checks proofs of them.
	APIKey *apikey.Service
	// Auth logs subjects in and checks the session tokens their requests carry.
	Auth *auth.Service
	// Ranger is for working with ranges.
	Ranger *ranger.Service
	// KV is for working with key-value pairs on ranges.
	KV *kv.Service
	// Project is for working with Projects.
	Project *project.Service
	// Bundle exports and imports projects as bundles.
	Bundle *bundle.Service
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
	// NI owns the NI chassis, devices, and stored configuration records of the NI task
	// types.
	NI *ni.Service
	// OPCUA owns the stored configuration records of the OPC UA task types.
	OPCUA *opcua.Service
	// LabJack owns the LabJack devices and the stored configuration records of the
	// LabJack task types.
	LabJack *labjack.Service
	// Modbus owns the stored configuration records of the Modbus task types.
	Modbus *modbus.Service
	// EtherCAT owns the stored configuration records of the EtherCAT task types.
	EtherCAT *ethercat.Service
	// HTTP owns the stored configuration records of the HTTP task types.
	HTTP *http.Service
	// Kafka owns the stored configuration records of the Kafka task types.
	Kafka *kafka.Service
	// TaskStopper stops running tasks ahead of a delete.
	TaskStopper *task.Stopper
	// ArcTask owns the stored configuration records of the arc task type.
	ArcTask *arctask.Service
	// RackTask owns the stored configuration records of the rack_status task type.
	RackTask *racktask.Service
	// PagerDuty owns the stored configuration records of the pagerduty_alert task
	// type.
	PagerDuty *pdruntime.Service
	// MQTT owns the stored configuration records of the MQTT task types.
	MQTT *mqtt.Service
	// Sift owns the stored configuration records of the Sift task types.
	Sift *sift.Service
	// Framer is for reading, writing, and streaming frames of telemetry from channels.
	Framer *framer.Service
	// Channel is the highest-level channel service and owns calculated channel
	// behavior.
	Channel *channel.Service
	// Control reads the control state of channels and publishes every transfer on the
	// control channel.
	Control *control.Service
	// License holds the Core's license and gates the API on it.
	License *license.Service
	// Arc is used for validating, saving, and executing arc automations.
	Arc *arc.Service
	// Metrics is used for collecting host machine metrics and publishing them over
	// channels
	Metrics *metrics.Service
	// Status is used for tracking the statuses
	Status *status.Service
	// View is used for managing views
	View *view.Service
	// Grant stores the grants that let a rack read secrets.
	Grant *grant.Service
	// Secret stores the hidden values that devices and tasks connect with.
	Secret *secret.Service
	// Certificate stores certificates and reports their expiry.
	Certificate *certificate.Service
	// ImEx is the central import/export registry.
	ImEx *imex.Service
	// Node serves the host node of the Core as a resource.
	Node *node.Service
	// Resource searches and names records of every type.
	Resource *resource.Service
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

func revokePasswordSessions(
	ctx context.Context,
	db *gorp.DB,
	passwords *password.Service,
	sessions *auth.Service,
	subject resource.ID,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		var p password.Password
		if err := passwords.NewRetrieve().
			Where(password.MatchSubjects(subject)).
			Entry(&p).
			Exec(ctx, tx); err != nil {
			return errors.Wrap(err, "load the root password")
		}
		return sessions.NewWriter(tx).RevokeCredential(ctx, p.Key)
	})
}

// OpenLayer opens the service layer using the provided configurations. Later
// configurations override the fields set in previous ones. If the configuration is
// invalid, or any services fail to open, Open returns a nil layer and an error. If the
// returned error is nil, the Layer must be closed by calling Close after use.
func OpenLayer(ctx context.Context, cfgs ...LayerConfig) (l *Layer, err error) {
	cfg, err := config.New(DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	l = &Layer{}
	cleanup, ok := service.NewOpener(ctx, &l.closer)
	defer func() { err = cleanup(err) }()
	// The migrations of the tables below read the ontology of an upgraded store, and
	// ontologyDropMigration deletes it once they have run.
	if err = ontology.Migrate(
		ctx,
		cfg.Distribution.DB,
		cfg.Child("ontology"),
	); !ok(err, nil) {
		return nil, err
	}
	if l.Group, err = group.OpenService(ctx, group.ServiceConfig{
		Instrumentation: cfg.Child("group"),
		DB:              cfg.Distribution.DB,
		// The visualization services open after the group service, so the hook reads
		// them when it runs.
		MoveMembers: func(
			ctx context.Context,
			tx gorp.Tx,
			groups []group.Key,
			top *resource.ID,
		) error {
			for _, move := range []group.MoveMembers{
				l.Schematic.MoveGroups,
				l.LinePlot.MoveGroups,
				l.Log.MoveGroups,
				l.Table.MoveGroups,
				l.Panel.MoveGroups,
			} {
				if err := move(ctx, tx, groups, top); err != nil {
					return err
				}
			}
			return nil
		},
	}); !ok(err, l.Group) {
		return nil, err
	}
	if l.Node, err = node.NewService(node.ServiceConfig{
		Instrumentation: cfg.Child("node"),
		Host:            cfg.Distribution.Host,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Label, err = label.OpenService(ctx, label.ServiceConfig{
		Instrumentation: cfg.Child("label"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.Label) {
		return nil, err
	}
	if l.Status, err = status.OpenService(
		ctx,
		status.ServiceConfig{
			Instrumentation: cfg.Child("status"),
			DB:              cfg.Distribution.DB,
		},
	); !ok(err, l.Status) {
		return nil, err
	}
	if l.License, err = license.OpenService(ctx, cfg.License, license.ServiceConfig{
		Instrumentation: cfg.Child("license"),
		DB:              cfg.Distribution.DB.KV(),
	}); !ok(err, l.License) {
		return nil, err
	}
	if l.Channel, err = channel.OpenService(ctx, channel.ServiceConfig{
		Instrumentation: cfg.Child("channel"),
		Channel:         cfg.Distribution.Channel,
		DB:              cfg.Distribution.DB,
		HostKey:         cfg.Distribution.Identity.HostKey,
		Limit:           l.License.CheckChannelLimit,
		ValidateNames:   cfg.ValidateChannelNames,
		Status:          l.Status,
	}); !ok(err, l.Channel) {
		return nil, err
	}
	channelGraph, err := calcgraph.Open(ctx, calcgraph.Config{
		Instrumentation: cfg.Child("channel.calculation.graph"),
		DB:              cfg.Distribution.DB,
		Channel:         l.Channel,
		Status:          l.Status,
	})
	if !ok(err, channelGraph) {
		return nil, err
	}
	if l.Signals, err = signals.New(signals.Config{
		DB:              cfg.Distribution.DB,
		Channel:         l.Channel,
		Framer:          cfg.Distribution.Framer,
		Instrumentation: cfg.Child("signals"),
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Framer, err = framer.OpenService(
		ctx,
		framer.ServiceConfig{
			Instrumentation: cfg.Child("framer"),
			DB:              cfg.Distribution.DB,
			Framer:          cfg.Distribution.Framer,
			Channel:         l.Channel,
			ChannelGraph:    channelGraph,
			Reserved:        l.Signals.Publishes,
		},
	); !ok(err, l.Framer) {
		return nil, err
	}
	if err = l.Signals.PublishTable(
		ctx,
		channelsignals.TableConfig(l.Channel),
	); !ok(err, nil) {
		return nil, err
	}
	if err = l.Signals.PublishActions(
		ctx,
		signals.ActionConfig[channel.Key, channel.Action]{
			Name:     "sy_channel_action_set",
			Actions:  l.Channel.Actions(),
			Resource: channel.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if closer, err := history.Publish(ctx, history.Config{
		Instrumentation: cfg.Child("framer_history"),
		Framer:          cfg.Distribution.Framer,
		Signals:         l.Signals,
	}); !ok(err, closer) {
		return nil, err
	}
	if closer, err := l.Signals.PublishJSON(
		ctx,
		signals.JSONPublisherConfig[string]{
			Observable: l.License,
			SetName:    license.SetChannelName,
		},
	); !ok(err, closer) {
		return nil, err
	}
	if err = l.Signals.PublishTable(
		ctx,
		signals.TableConfig[group.Key, group.Group]{
			Changes:  l.Group.Changes,
			Resource: group.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if err = l.Signals.PublishActions(
		ctx,
		signals.ActionConfig[group.Key, group.Action]{
			Name:     "sy_group_action_set",
			Actions:  l.Group.Actions(),
			Resource: group.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if err = l.Signals.PublishTable(
		ctx,
		signals.TableConfig[label.Key, label.Label]{
			Changes:  l.Label.Changes,
			Resource: label.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if err = l.Signals.PublishActions(
		ctx,
		signals.ActionConfig[label.Key, label.Action]{
			Name:     "sy_label_action_set",
			Actions:  l.Label.Actions(),
			Resource: label.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if err = l.Signals.PublishTable(
		ctx,
		signals.TableConfig[status.Key, status.Status[any]]{
			Changes:  l.Status.Changes,
			Resource: status.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	// The Driver reads sy_status_set as statuses, so actions take their own channel.
	if err = l.Signals.PublishActions(
		ctx,
		signals.ActionConfig[status.Key, status.Action]{
			Name:     "sy_status_action_set",
			Actions:  l.Status.Actions(),
			Resource: status.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if l.Control, err = control.OpenService(ctx, control.ServiceConfig{
		Instrumentation: cfg.Child("control"),
		Control:         cfg.Distribution.Control,
		Signals:         l.Signals,
	}); !ok(err, l.Control) {
		return nil, err
	}
	if err = passwordversions.MoveLegacy(
		ctx,
		cfg.Distribution.DB,
		cfg.Child("password"),
	); !ok(err, nil) {
		return nil, err
	}
	// Users and racks reference roles, and racks take the Driver role, so roles open
	// first.
	roles, err := rbac.OpenRoles(ctx, rbac.RolesConfig{
		Instrumentation: cfg.Child("rbac"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
	})
	if !ok(err, roles) {
		return nil, err
	}
	if err = l.Signals.PublishActions(
		ctx,
		signals.ActionConfig[role.Key, role.Action]{
			Name:     "sy_role_action_set",
			Actions:  roles.Role.Actions(),
			Resource: role.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if err = l.Signals.PublishActions(
		ctx,
		signals.ActionConfig[policy.Key, policy.Action]{
			Name:     "sy_policy_action_set",
			Actions:  roles.Policy.Actions(),
			Resource: policy.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if l.Password, err = password.OpenService(ctx, password.ServiceConfig{
		Instrumentation: cfg.Child("password"),
		DB:              cfg.Distribution.DB,
		FIPS:            cfg.FIPS,
	}); !ok(err, l.Password) {
		return nil, err
	}
	if l.User, err = user.OpenService(ctx, user.ServiceConfig{
		Instrumentation: cfg.Child("user"),
		DB:              cfg.Distribution.DB,
		Password:        l.Password,
		Signals:         l.Signals,
	}); !ok(err, l.User) {
		return nil, err
	}
	if err = l.Signals.PublishActions(
		ctx,
		signals.ActionConfig[user.Key, user.Action]{
			Name:     "sy_user_action_set",
			Actions:  l.User.Actions(),
			Resource: user.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if l.APIKey, err = apikey.OpenService(ctx, apikey.ServiceConfig{
		Instrumentation: cfg.Child("apikey"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.APIKey) {
		return nil, err
	}
	if l.Rack, err = rack.OpenService(ctx, rack.ServiceConfig{
		Instrumentation: cfg.Child("rack"),
		DB:              cfg.Distribution.DB,
		HostKey:         cfg.Distribution.Identity.HostKey,
		Status:          l.Status,
		DriverRole:      roles.Builtin.DriverKey,
	}); !ok(err, l.Rack) {
		return nil, err
	}
	if err = l.Signals.PublishTable(
		ctx,
		signals.TableConfig[rack.Key, rack.Rack]{
			Changes:  l.Rack.Changes,
			Resource: rack.Key.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	// The Driver reads sy_rack_set as racks, so actions take their own channel.
	if err = l.Signals.PublishActions(
		ctx,
		signals.ActionConfig[rack.Key, rack.Action]{
			Name:     "sy_rack_action_set",
			Actions:  l.Rack.Actions(),
			Resource: rack.Key.ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if l.RBAC, err = rbac.NewService(ctx, rbac.ServiceConfig{
		Instrumentation: cfg.Child("rbac"),
		DB:              cfg.Distribution.DB,
		Roles:           roles,
		User:            l.User,
		Rack:            l.Rack,
	}); !ok(err, nil) {
		return nil, err
	}
	// The RBAC migrations give each stored user a role, so the root user is provisioned
	// after them.
	var (
		root                user.User
		rootPasswordChanged bool
	)
	if !cfg.RootCredentials.IsZero() {
		if root, rootPasswordChanged, err = user.ProvisionRoot(
			ctx,
			l.User,
			l.Password,
			cfg.RootCredentials,
			roles.Builtin.OwnerKey,
		); !ok(err, nil) {
			return nil, err
		}
		if rootPasswordChanged {
			cfg.L.Info(
				"rotated root user password to match the start settings",
				zap.String("username", cfg.RootCredentials.Username),
			)
		}
	}
	if l.Auth, err = auth.OpenService(ctx, auth.ServiceConfig{
		Instrumentation: cfg.Child("auth"),
		DB:              cfg.Distribution.DB,
		Authenticators: map[auth.Method]auth.Authenticator{
			auth.MethodPassword: l.Password,
			auth.MethodAPIKey:   l.APIKey,
		},
		IdleTimeout: cfg.SessionIdleTimeout,
	}); !ok(err, l.Auth) {
		return nil, err
	}
	if rootPasswordChanged {
		if err = revokePasswordSessions(
			ctx,
			cfg.Distribution.DB,
			l.Password,
			l.Auth,
			root.ResourceID(),
		); !ok(err, nil) {
			return nil, err
		}
	}
	if l.Ranger, err = ranger.OpenService(ctx, ranger.ServiceConfig{
		Instrumentation: cfg.Child("ranger"),
		DB:              cfg.Distribution.DB,
		Group:           l.Group,
		Signals:         l.Signals,
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
	// The visualization migrations read the projects converted from workspaces, so the
	// project table opens first.
	if l.Project, err = project.OpenService(ctx, project.ServiceConfig{
		Instrumentation: cfg.Child("project"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
	}); !ok(err, l.Project) {
		return nil, err
	}
	l.ImEx = imex.NewService()
	if l.Schematic, err = schematic.OpenService(ctx, schematic.ServiceConfig{
		Instrumentation: cfg.Child("schematic"),
		DB:              cfg.Distribution.DB,
		Group:           l.Group,
		Signals:         l.Signals,
		ImEx:            l.ImEx,
	}); !ok(err, l.Schematic) {
		return nil, err
	}
	if l.LinePlot, err = lineplot.OpenService(ctx, lineplot.ServiceConfig{
		Instrumentation: cfg.Child("lineplot"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
		ImEx:            l.ImEx,
	}); !ok(err, l.LinePlot) {
		return nil, err
	}
	if l.Log, err = log.OpenService(ctx, log.ServiceConfig{
		Instrumentation: cfg.Child("log"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
		ImEx:            l.ImEx,
	}); !ok(err, l.Log) {
		return nil, err
	}
	if l.Panel, err = panel.OpenService(ctx, panel.ServiceConfig{
		Instrumentation: cfg.Child("panel"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
	}); !ok(err, l.Panel) {
		return nil, err
	}
	if l.Table, err = table.OpenService(ctx, table.ServiceConfig{
		Instrumentation: cfg.Child("table"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
		ImEx:            l.ImEx,
	}); !ok(err, l.Table) {
		return nil, err
	}
	if l.Grant, err = grant.OpenService(ctx, grant.ServiceConfig{
		Instrumentation: cfg.Child("grant"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.Grant) {
		return nil, err
	}
	if l.Secret, err = secret.OpenService(ctx, secret.ServiceConfig{
		Instrumentation: cfg.Child("secret"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
		Grant:           l.Grant,
		EncryptionKey:   cfg.SecretsKey,
	}); !ok(err, l.Secret) {
		return nil, err
	}
	if l.Device, err = device.OpenService(ctx, device.ServiceConfig{
		Instrumentation: cfg.Child("device"),
		DB:              cfg.Distribution.DB,
		Signals:         l.Signals,
		Status:          l.Status,
		Rack:            l.Rack,
		References: []gorp.Reference[device.Key, device.Device]{
			sift.APIKeyReference(),
		},
	}); !ok(err, l.Device) {
		return nil, err
	}
	if l.NI, err = ni.OpenService(ctx, ni.ServiceConfig{
		Instrumentation: cfg.Child("ni"),
		DB:              cfg.Distribution.DB,
		Status:          l.Status,
		Signals:         l.Signals,
	}); !ok(err, l.NI) {
		return nil, err
	}
	if l.OPCUA, err = opcua.OpenService(ctx, opcua.ServiceConfig{
		Instrumentation: cfg.Child("opcua"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.OPCUA) {
		return nil, err
	}
	if l.LabJack, err = labjack.OpenService(ctx, labjack.ServiceConfig{
		Instrumentation: cfg.Child("labjack"),
		DB:              cfg.Distribution.DB,
		Status:          l.Status,
		Signals:         l.Signals,
	}); !ok(err, l.LabJack) {
		return nil, err
	}
	if l.Modbus, err = modbus.OpenService(ctx, modbus.ServiceConfig{
		Instrumentation: cfg.Child("modbus"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.Modbus) {
		return nil, err
	}
	if l.EtherCAT, err = ethercat.OpenService(ctx, ethercat.ServiceConfig{
		Instrumentation: cfg.Child("ethercat"),
		DB:              cfg.Distribution.DB,
		Device:          l.Device,
	}); !ok(err, l.EtherCAT) {
		return nil, err
	}
	if l.HTTP, err = http.OpenService(ctx, http.ServiceConfig{
		Instrumentation: cfg.Child("http"),
		DB:              cfg.Distribution.DB,
		Channel:         l.Channel,
	}); !ok(err, l.HTTP) {
		return nil, err
	}
	if l.Kafka, err = kafka.OpenService(ctx, kafka.ServiceConfig{
		Instrumentation: cfg.Child("kafka"),
		DB:              cfg.Distribution.DB,
		Channel:         l.Channel,
	}); !ok(err, l.Kafka) {
		return nil, err
	}
	// Arc task configs reference Arcs, and the task migrations write them.
	arcTable, err := arc.OpenTable(ctx, arc.TableConfig{
		Instrumentation: cfg.Child("arc"),
		DB:              cfg.Distribution.DB,
		Parser: parser.Config{
			AllowDashedNames: !l.Channel.ShouldValidateNames(),
		},
	})
	if !ok(err, arcTable) {
		return nil, err
	}
	if l.ArcTask, err = arctask.OpenService(ctx, arctask.ServiceConfig{
		Instrumentation: cfg.Child("arc_task"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.ArcTask) {
		return nil, err
	}
	if l.RackTask, err = racktask.OpenService(ctx, racktask.ServiceConfig{
		Instrumentation: cfg.Child("rack_task"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.RackTask) {
		return nil, err
	}
	if l.PagerDuty, err = pdruntime.OpenService(ctx, pdruntime.ServiceConfig{
		Instrumentation: cfg.Child("pagerduty"),
		DB:              cfg.Distribution.DB,
		Secret:          l.Secret,
	}); !ok(err, l.PagerDuty) {
		return nil, err
	}
	if l.MQTT, err = mqtt.OpenService(ctx, mqtt.ServiceConfig{
		Instrumentation: cfg.Child("mqtt"),
		DB:              cfg.Distribution.DB,
		Channel:         l.Channel,
	}); !ok(err, l.MQTT) {
		return nil, err
	}
	if l.Sift, err = sift.OpenService(ctx, sift.ServiceConfig{
		Instrumentation: cfg.Child("sift"),
		DB:              cfg.Distribution.DB,
	}); !ok(err, l.Sift) {
		return nil, err
	}
	configStores := slices.Concat(
		l.NI.Stores(), l.OPCUA.Stores(), l.LabJack.Stores(), l.Modbus.Stores(),
		l.EtherCAT.Stores(), l.HTTP.Stores(), l.Kafka.Stores(), l.ArcTask.Stores(),
		l.RackTask.Stores(), l.PagerDuty.Stores(), l.MQTT.Stores(), l.Sift.Stores(),
	)
	taskConfigs, err := taskconfig.NewRegistry(configStores...)
	if !ok(err, nil) {
		return nil, err
	}
	if l.Task, err = task.OpenService(ctx, task.ServiceConfig{
		Instrumentation: cfg.Child("task"),
		DB:              cfg.Distribution.DB,
		Channel:         l.Channel,
		Rack:            l.Rack,
		Status:          l.Status,
		Signals:         l.Signals,
		ImEx:            l.ImEx,
		Configs:         taskConfigs,
		MatchArcs:       l.ArcTask.MatchArcs,
		// An Arc task holds a key and hash pointing at an Arc document, so it has no
		// file form of its own, and the Arc service owns the "arc" file type.
		ImExExcluded: []string{arctask.Type},
	}); !ok(err, l.Task) {
		return nil, err
	}
	if l.TaskStopper, err = task.NewStopper(task.StopperConfig{
		DB:     cfg.Distribution.DB,
		Task:   l.Task,
		Status: l.Status,
		Framer: l.Framer,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Arc, err = arc.OpenService(
		ctx,
		arc.ServiceConfig{
			Instrumentation: cfg.Child("arc"),
			DB:              cfg.Distribution.DB,
			Table:           arcTable,
			Channel:         l.Channel,
			Task:            l.Task,
			MatchTasks:      l.ArcTask.MatchArcs,
			Status:          l.Status,
			Signals:         l.Signals,
			ImEx:            l.ImEx,
		},
	); !ok(err, l.Arc) {
		return nil, err
	}
	if l.View, err = view.OpenService(
		ctx,
		view.ServiceConfig{
			Instrumentation: cfg.Child("view"),
			DB:              cfg.Distribution.DB,
			Signals:         l.Signals,
		},
	); !ok(err, l.View) {
		return nil, err
	}
	if l.Certificate, err = certificate.OpenService(ctx, certificate.ServiceConfig{
		Instrumentation: cfg.Child("certificate"),
		DB:              cfg.Distribution.DB,
		Secret:          l.Secret,
		Status:          l.Status,
		Signals:         l.Signals,
	}); !ok(err, l.Certificate) {
		return nil, err
	}
	if l.Resource, err = resource.NewService(resource.ServiceConfig{
		Providers: map[resource.Type]resource.Provider{
			resource.TypeArc:             l.Arc.Resources(),
			resource.TypeArcDirectory:    l.Arc.DirectoryResources(),
			resource.TypeCertificate:     l.Certificate.Resources(),
			resource.TypeChannel:         l.Channel.Resources(),
			resource.TypeDevice:          l.Device.Resources(),
			resource.TypeGroup:           l.Group.Resources(),
			resource.TypeLabel:           l.Label.Resources(),
			resource.TypeLabjackDevice:   l.LabJack.Devices.Resources(),
			resource.TypeLineplot:        l.LinePlot.Resources(),
			resource.TypeLog:             l.Log.Resources(),
			resource.TypeNiChassis:       l.NI.Chassis.Resources(),
			resource.TypeNiDevice:        l.NI.Devices.Resources(),
			resource.TypeNode:            l.Node.Resources(),
			resource.TypePanel:           l.Panel.Resources(),
			resource.TypePolicy:          l.RBAC.Policy.Resources(),
			resource.TypeProject:         l.Project.Resources(),
			resource.TypeRack:            l.Rack.Resources(),
			resource.TypeRange:           l.Ranger.Resources(),
			resource.TypeRole:            l.RBAC.Role.Resources(),
			resource.TypeSchematic:       l.Schematic.Resources(),
			resource.TypeSchematicSymbol: l.Schematic.Symbol.Resources(),
			resource.TypeSecret:          l.Secret.Resources(),
			resource.TypeStatus:          l.Status.Resources(),
			resource.TypeTable:           l.Table.Resources(),
			resource.TypeTask:            l.Task.Resources(),
			resource.TypeUser:            l.User.Resources(),
			resource.TypeView:            l.View.Resources(),
		},
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Bundle, err = bundle.NewService(bundle.Config{
		Resource: l.Resource,
		Project:  l.Project,
		Group:    l.Group,
		Panel:    l.Panel,
		ImEx:     l.ImEx,
	}); !ok(err, nil) {
		return nil, err
	}
	if l.Metrics, err = metrics.OpenService(
		ctx,
		metrics.ServiceConfig{
			Instrumentation: cfg.Child("metrics"),
			DB:              cfg.Distribution.DB,
			Framer:          l.Framer,
			Channel:         l.Channel,
			HostKey:         cfg.Distribution.Identity.HostKey,
			Storage:         cfg.Storage,
			Group:           l.Group,
		},
	); !ok(err, l.Metrics) {
		return nil, err
	}
	// Composition migrations move data across service boundaries, so they can only run
	// once every service table above is open: a table's own chain runs at open, before
	// later services have staged the legacy data these migrations consume. Table chains
	// handle single-table format upgrades; anything that reads another service's staged
	// data belongs here.
	if err = gorp.Migrate(ctx, gorp.MigrateConfig{
		Instrumentation: cfg.Child("composition"),
		DB:              cfg.Distribution.DB,
		Namespace:       "Composition",
		Migrations: append(
			l.Panel.CompositionMigrations(),
			ontologyDropMigration,
			l.Group.BuiltinDeleteMigration(),
			rangeAliasDropMigration,
		),
	}); !ok(err, nil) {
		return nil, err
	}
	if err = deleteRetiredChannels(
		ctx,
		cfg.Distribution.DB,
		l.Channel,
	); !ok(err, nil) {
		return nil, err
	}
	// The change channels send no commit before Start, so the migrations above send
	// nothing.
	signalsCloser, err := l.Signals.Start(ctx)
	if !ok(err, signalsCloser) {
		return nil, err
	}
	arcFactory, err := arctask.NewFactory(arctask.FactoryConfig{
		Instrumentation: cfg.Child("arc.task"),
		DB:              cfg.Distribution.DB,
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
		DB:              cfg.Distribution.DB,
		Status:          l.Status,
		Secret:          l.Secret,
	})
	if !ok(err, nil) {
		return nil, err
	}
	kafkaFactory, err := kafka.NewFactory(kafka.FactoryConfig{
		Instrumentation: cfg.Child("kafka"),
		DB:              cfg.Distribution.DB,
		Device:          l.Device,
		Channel:         l.Channel,
		Framer:          l.Framer,
		Status:          l.Status,
	})
	if !ok(err, nil) {
		return nil, err
	}
	mqttFactory, err := mqtt.OpenFactory(mqtt.FactoryConfig{
		Instrumentation: cfg.Child("mqtt"),
		DB:              cfg.Distribution.DB,
		Device:          l.Device,
		Channel:         l.Channel,
		Framer:          l.Framer,
		Status:          l.Status,
	})
	if !ok(err, mqttFactory) {
		return nil, err
	}
	siftFactory, err := sift.OpenFactory(sift.FactoryConfig{
		Instrumentation: cfg.Child("sift"),
		DB:              cfg.Distribution.DB,
		Device:          l.Device,
		Status:          l.Status,
		Secret:          l.Secret,
		Ranger:          l.Ranger,
		Channel:         l.Channel,
		Framer:          l.Framer,
		Label:           l.Label,
		Task:            l.Task,
	})
	if !ok(err, siftFactory) {
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
		Factories: []driver.Factory{
			arcFactory,
			pdFactory,
			kafkaFactory,
			mqttFactory,
			siftFactory,
		},
		HostKey: cfg.Distribution.Identity.HostKey,
	}); !ok(err, l.Driver) {
		return nil, err
	}
	return l, nil
}
