// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil creates tasks for the specs of packages that store task configs.
package testutil

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
)

// TasksConfig is the configuration for opening Tasks.
type TasksConfig struct {
	// DB is the database the tasks are stored in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology holds the task resources. OpenTasks opens one on DB when Rack is nil.
	//
	// [OPTIONAL]
	Ontology *ontology.Ontology
	// Status stores the task statuses. OpenTasks opens one on DB when Rack is nil.
	//
	// [OPTIONAL]
	Status *status.Service
	// Rack holds the racks the tasks deploy to. OpenTasks opens one on DB when nil.
	//
	// [OPTIONAL]
	Rack *rack.Service
	// Stores keep the configs of the tasks.
	//
	// [REQUIRED]
	Stores []config.Store
	// MatchArcs filters tasks by the Arcs they run.
	//
	// [OPTIONAL]
	MatchArcs func(arcs ...uuid.UUID) task.Filter
}

// Tasks creates tasks through a task service.
type Tasks struct {
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

func (t *Tasks) open(ctx context.Context, cfg TasksConfig) error {
	if cfg.Rack == nil {
		if err := t.openRack(ctx, &cfg); err != nil {
			return err
		}
	}
	configs, err := config.NewRegistry(cfg.Stores...)
	if err != nil {
		return err
	}
	if t.svc, err = task.OpenService(ctx, task.ServiceConfig{
		DB:        cfg.DB,
		Ontology:  cfg.Ontology,
		Rack:      cfg.Rack,
		Status:    cfg.Status,
		ImEx:      imex.NewService(),
		Configs:   configs,
		MatchArcs: cfg.MatchArcs,
	}); err != nil {
		return err
	}
	t.closer = append(t.closer, t.svc)
	return nil
}

func (t *Tasks) openRack(ctx context.Context, cfg *TasksConfig) (err error) {
	if cfg.Ontology, err = ontology.Open(ctx, ontology.Config{DB: cfg.DB}); err != nil {
		return err
	}
	t.closer = append(t.closer, cfg.Ontology)
	if cfg.Status, err = status.OpenService(ctx, status.ServiceConfig{
		DB:       cfg.DB,
		Ontology: cfg.Ontology,
	}); err != nil {
		return err
	}
	t.closer = append(t.closer, cfg.Status)
	role, err := racktestutil.CreateDriverRole(ctx, cfg.Ontology)
	if err != nil {
		return err
	}
	if cfg.Rack, err = rack.OpenService(ctx, rack.ServiceConfig{
		DB:           cfg.DB,
		Ontology:     cfg.Ontology,
		HostProvider: testutil.NewStaticHostProvider(1),
		Status:       cfg.Status,
		DriverRole:   role,
	}); err != nil {
		return err
	}
	t.closer = append(t.closer, cfg.Rack)
	return nil
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

// NewRetrieve opens a query for the tasks.
func (t *Tasks) NewRetrieve() task.Retrieve { return t.svc.NewRetrieve() }

// Close closes the services that OpenTasks opened.
func (t *Tasks) Close() error { return t.closer.Close() }
