// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil creates tasks, and the devices and statuses their configs reference,
// for the specs of packages that store task configs.
package testutil

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
)

// TasksConfig is the configuration for opening Tasks.
type TasksConfig struct {
	// DB is the database the tasks are stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Status stores the task statuses. OpenTasks opens one on DB when it is nil.
	//
	// [OPTIONAL]
	Status *status.Service
	// Rack holds the racks the tasks deploy to. OpenTasks opens one on DB when nil.
	//
	// [OPTIONAL]
	Rack *rack.Service
	// Device stores the devices that configs reference. OpenTasks opens one on DB when
	// it is nil.
	//
	// [OPTIONAL]
	Device *device.Service
	// Stores keep the configs of the tasks.
	//
	// [REQUIRED]
	Stores []config.Store
}

// Tasks creates tasks through a task service, and the targets their configs reference.
type Tasks struct {
	*Targets
	svc    *task.Service
	closer xio.MultiCloser
}

// OpenTasks opens a task service with cfg. Close the returned Tasks to close the
// services it opens.
func OpenTasks(ctx context.Context, cfg TasksConfig) (*Tasks, error) {
	t := &Tasks{}
	if err := t.open(ctx, cfg); err != nil {
		return nil, errors.Join(err, t.closer.Close())
	}
	return t, nil
}

func (t *Tasks) open(ctx context.Context, cfg TasksConfig) (err error) {
	if cfg.Status == nil {
		if cfg.Status, err = status.OpenService(ctx, status.ServiceConfig{
			DB: cfg.DB,
		}); err != nil {
			return err
		}
		t.closer = append(t.closer, cfg.Status)
	}
	if cfg.Rack == nil {
		if err := t.openRack(ctx, &cfg); err != nil {
			return err
		}
	}
	if cfg.Device == nil {
		if err := t.openDevice(ctx, &cfg); err != nil {
			return err
		}
	}
	t.Targets = NewTargets(TargetsConfig{
		Device: cfg.Device,
		Status: cfg.Status,
		Rack:   cfg.Rack.EmbeddedKey,
	})
	configs, err := config.NewRegistry(cfg.Stores...)
	if err != nil {
		return err
	}
	if t.svc, err = task.OpenService(ctx, task.ServiceConfig{
		DB:      cfg.DB,
		Rack:    cfg.Rack,
		Status:  cfg.Status,
		ImEx:    imex.NewService(),
		Configs: configs,
	}); err != nil {
		return err
	}
	t.closer = append(t.closer, t.svc)
	return nil
}

func (t *Tasks) openRack(ctx context.Context, cfg *TasksConfig) (err error) {
	roles, err := role.OpenService(ctx, role.ServiceConfig{
		DB: cfg.DB,
	})
	if err != nil {
		return err
	}
	t.closer = append(t.closer, roles)
	driverRole, err := racktestutil.CreateDriverRole(ctx, roles)
	if err != nil {
		return err
	}
	if cfg.Rack, err = rack.OpenService(ctx, rack.ServiceConfig{
		DB:           cfg.DB,
		HostProvider: testutil.NewStaticHostProvider(1),
		Status:       cfg.Status,
		DriverRole:   driverRole,
	}); err != nil {
		return err
	}
	t.closer = append(t.closer, cfg.Rack)
	return nil
}

func (t *Tasks) openDevice(ctx context.Context, cfg *TasksConfig) (err error) {
	if cfg.Device, err = device.OpenService(ctx, device.ServiceConfig{
		DB:     cfg.DB,
		Status: cfg.Status,
		Rack:   cfg.Rack,
	}); err != nil {
		return err
	}
	t.closer = append(t.closer, cfg.Device)
	return nil
}

// TargetsConfig is the configuration for Targets.
type TargetsConfig struct {
	// Device and Status store the devices and statuses that Targets creates.
	//
	// [REQUIRED]
	Device *device.Service
	Status *status.Service
	// Rack is the rack that Targets creates devices on.
	//
	// [REQUIRED]
	Rack rack.Key
}

// Targets creates the devices and statuses that task configs reference.
type Targets struct{ cfg TargetsConfig }

// NewTargets returns Targets that create devices and statuses through the services in
// cfg.
func NewTargets(cfg TargetsConfig) *Targets { return &Targets{cfg: cfg} }

// CreateDevice creates a device with key on the configured rack for configs to
// reference.
func (t *Targets) CreateDevice(ctx context.Context, key device.Key) error {
	return t.cfg.Device.NewWriter(nil).Create(ctx, &device.Device{
		Key:      key,
		Rack:     t.cfg.Rack,
		Name:     string(key),
		Make:     "test",
		Model:    "test",
		Location: "test",
	})
}

// CreateStatus creates a status with key for configs to reference.
func (t *Targets) CreateStatus(ctx context.Context, key status.Key) error {
	return t.cfg.Status.NewWriter(nil).Set(ctx, &status.Status[any]{
		Key:     key,
		Name:    string(key),
		Variant: status.VariantInfo,
		Time:    telem.Now(),
	})
}

// Create creates a task with the config data in store. It returns the key of the task,
// which is the key of its config record.
func (t *Tasks) Create(
	ctx context.Context,
	store config.Store,
	data msgpack.EncodedJSON,
) (task.Key, error) {
	tsk := task.Task{Name: store.Type(), Type: store.Type(), Config: data}
	if err := t.svc.NewWriter(nil).Create(ctx, &tsk); err != nil {
		return uuid.Nil(), err
	}
	return tsk.Key, nil
}

// Close closes the services that OpenTasks opened.
func (t *Tasks) Close() error { return t.closer.Close() }
