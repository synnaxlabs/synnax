// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening an ICD Service.
type ServiceConfig struct {
	// Table stores the ICDs.
	//
	// [REQUIRED]
	Table *Table
	// Task rewrites the tasks that use an ICD when the messages they use change.
	//
	// [REQUIRED]
	Task *task.Service
	// Uses returns the messages, and the fields of each, that a task config uses.
	//
	// [REQUIRED]
	Uses func(config msgpack.EncodedJSON) ([]Use, error)
	// ImEx is the import/export registry the service registers itself with.
	//
	// [REQUIRED]
	ImEx *imex.Service
	// Signals publishes ICD changes to clients.
	//
	// [OPTIONAL] - Defaults to nil.
	Signals *signals.Provider
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Table = override.Nil(c.Table, other.Table)
	c.Task = override.Nil(c.Task, other.Task)
	c.Uses = override.Nil(c.Uses, other.Uses)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("icd")
	v.NotNil("table", c.Table)
	v.NotNil("task", c.Task)
	v.NotNil("uses", c.Uses)
	v.NotNil("imex", c.ImEx)
	return v.Error()
}

// Service creates, retrieves, and deletes ICDs.
type Service struct {
	cfg    ServiceConfig
	table  *Table
	closer xio.MultiCloser
}

// OpenService opens a Service with the provided configuration. The service must be
// closed by calling Close.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	s = &Service{}
	if s.cfg, err = config.New(ServiceConfig{}, cfgs...); err != nil {
		return nil, err
	}
	s.table = s.cfg.Table
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if err = s.cfg.ImEx.RegisterImportExporter(s); !ok(err, nil) {
		return nil, err
	}
	if s.cfg.Signals == nil {
		return s, nil
	}
	if err = s.cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, ICD]{
		Changes:  s.table.table.Changes,
		Resource: ResourceID,
	}); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

// Close releases the resources held by the service. Close is not safe to call
// concurrently with any other service method.
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a Writer that executes against tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{reader: s.reader(tx), tx: tx}
}

// Check returns every problem that would stop Writer.Create from storing ic, read
// through snap, without storing it: each problem the package-level Check finds, and
// each message or field that a task uses and ic lacks.
func (s *Service) Check(
	ctx context.Context,
	snap gorp.Snapshot,
	ic ICD,
) (library.Problems, error) {
	return s.reader(snap).check(ctx, ic)
}

func (s *Service) reader(snap gorp.Snapshot) reader {
	return reader{snap: snap, table: s.table, tasks: s.cfg.Task, uses: s.cfg.Uses}
}

// NewRetrieve opens a query for ICDs.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.table.NewRetrieve(), indexes: s.table.indexes}
}
