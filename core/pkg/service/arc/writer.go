// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"encoding/json/v2"
	"maps"
	"slices"
	"uuid"

	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/arc/drift"
	taskversions "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// TaskType is the task type for tasks that execute an Arc module.
const TaskType = "arc"

// Writer is used to create, update, and delete arcs within Synnax. The writer executes
// all operations within the transaction provided to the Service.NewWriter method.
type Writer struct {
	tx          gorp.Tx
	root        directory.Key
	directories *directory.Table
	reserved    set.Set[string]
	tasks       *task.Service
	matchTasks  func(arcs ...Key) task.Filter
	status      *status.Service
	table       *gorp.Table[Key, Arc]
	drifts      *gorp.Table[drift.Key, drift.Record]
	dispatcher  actions.Dispatcher[Key, Action]
	sweeper     textSweeper
	resyncs     *gorp.Attachment[Key]
	deletes     *gorp.Attachment[Key]
	// directoryDispatcher attaches the actions that change directories.
	directoryDispatcher actions.Dispatcher[directory.Key, directory.Action]
	editedModules       *gorp.Attachment[Key]
	pathChanges         *gorp.Attachment[struct{}]
	rebuild             func(
		context.Context, gorp.Snapshot, *channel.Lookups, Key,
	) (Build, error)
	hashModules func(
		context.Context, gorp.Snapshot, []string,
	) (map[string]string, error)
	now func() telem.TimeStamp
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	w.directoryDispatcher = w.directoryDispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given Arc in the root directory. If the Arc does not have a key, a
// new key will be generated. Create returns an error wrapping validate.ErrValidation
// when an Arc with the key exists.
func (w Writer) Create(ctx context.Context, a *Arc) error {
	if a.Variant == nil {
		return errors.Wrap(
			validate.ErrValidation, "an Arc must be an automation or a module",
		)
	}
	b := a.Base()
	if b.Key == uuid.Nil() {
		b.Key = uuid.New()
	}
	// A decode defaults an omitted source, but an explicit null clears it.
	if b.Source.Variant == nil {
		return errors.Wrap(validate.ErrValidation, "source: required")
	}
	if src, ok := b.Source.Variant.(TextSource); ok &&
		len(src.Text.Doc.Inserts) == 0 && src.Text.Raw != "" {
		src.Text.Doc = text.Create(src.Text.Raw)
		b.Source.Variant = src
	}
	a.SetBase(b)
	if err := a.Validate(); err != nil {
		return err
	}
	module := a.Type() == TypeModule
	if err := w.validateEntry(ctx, w.root, a.ResourceID(), b.Name, module); err != nil {
		return err
	}
	b.Directory = &w.root
	a.SetBase(b)
	if err := w.table.NewCreate().Exclusive().Entry(a).Exec(ctx, w.tx); err != nil {
		return err
	}
	if module {
		w.pathChanges.Add(w.tx, struct{}{})
		w.resyncs.Add(w.tx, b.Key)
	}
	w.dispatcher.Attach(
		w.tx,
		b.Key,
		NewCreateAction(CreatePayload{Arc: *a}),
	)
	return nil
}

// CreateMany creates the given Arcs. It returns an error wrapping
// validate.ErrValidation when an Arc with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, arcs *[]Arc) error {
	for i := range *arcs {
		if err := w.Create(ctx, &(*arcs)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the Arcs with the given keys. If the Arc has child tasks, those tasks
// will also be deleted.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	for _, key := range keys {
		if err := w.deleteChildTasks(ctx, key); err != nil {
			return err
		}
	}
	module, err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Arc](keys...)).
		Where(gorp.Match(func(_ gorp.Context, a *Arc) (bool, error) {
			return a.Type() == TypeModule, nil
		})).
		SkipMissingKeys().
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if err = w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Arc](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.deletes.Add(w.tx, keys...)
	if module {
		w.pathChanges.Add(w.tx, struct{}{})
	}
	return nil
}

func (w Writer) deleteChildTasks(ctx context.Context, key Key) error {
	taskKeys, err := retrieveTaskKeys(ctx, w.tasks, w.tx, w.matchTasks(key))
	if err != nil {
		return err
	}
	tw := w.tasks.NewWriter(w.tx)
	for _, taskKey := range taskKeys {
		if err = tw.Delete(ctx, taskKey, false); err != nil {
			return err
		}
	}
	return nil
}

func retrieveTaskKeys(
	ctx context.Context,
	tasks *task.Service,
	snap gorp.Snapshot,
	f task.Filter,
) ([]task.Key, error) {
	var matched []task.Task
	if err := tasks.NewRetrieve().
		Where(f).
		Entries(&matched).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	keys := make([]task.Key, len(matched))
	for i, t := range matched {
		keys[i] = t.Key
	}
	return keys, nil
}

// TaskUpdate holds the fields of an arc's task to change. A nil field stays as it is.
type TaskUpdate struct {
	// Rack binds the arc to the rack, creating its task or moving the existing one. A
	// zero rack unbinds the arc, deleting its task.
	Rack *rack.Key
	// Performance sets how closely the task's runtime holds timer deadlines.
	Performance *taskversions.Performance
	// Build is the arc's compile from Service.Build. UpdateTask compiles the arc in its
	// transaction instead when Build is zero. When a write changed what Build read,
	// UpdateTask still uses Build and schedules a resync that compiles it again.
	Build Build
}

// UpdateTask changes the task of the arc with the given key. It returns the task, or
// nil after an unbind. It returns an error wrapping validate.ErrValidation when
// unbinding a running task, when unbinding and setting the performance in one update,
// or when setting the performance of an arc with no task.
func (w Writer) UpdateTask(
	ctx context.Context,
	key Key,
	update TaskUpdate,
) (*task.Task, error) {
	existing, err := retrieveTaskKeys(ctx, w.tasks, w.tx, w.matchTasks(key))
	if err != nil {
		return nil, err
	}
	if update.Rack != nil && *update.Rack == 0 {
		if update.Performance != nil {
			return nil, errors.Wrap(
				validate.ErrValidation,
				"cannot set the performance of an arc while unbinding it",
			)
		}
		return nil, w.clearRack(ctx, existing)
	}
	var (
		a   Arc
		cfg taskversions.Config
		tsk task.Task
	)
	if err = w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Arc](key)).
		Entry(&a).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	if a.Type() == TypeModule {
		return nil, errors.Wrap(
			validate.ErrValidation, "a module cannot be bound to a rack",
		)
	}
	if len(existing) > 0 {
		if tsk, cfg, err = w.retrieveTask(ctx, existing[0]); err != nil {
			return nil, err
		}
	} else if update.Rack == nil {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"cannot set the performance of an arc with no rack",
		)
	} else {
		cfg.SetDefaults()
	}
	if update.Rack != nil {
		tsk.Rack = *update.Rack
	}
	if update.Performance != nil {
		cfg.Performance = *update.Performance
	}
	rec, err := w.record(ctx, tsk.Key)
	if err != nil {
		return nil, err
	}
	b, err := w.current(ctx, update.Build, key)
	if err != nil {
		return nil, err
	}
	var hashes drift.Hashes
	if cfg.Hash, hashes, err = w.hash(ctx, a, rec.Stored, b); err != nil {
		return nil, err
	}
	written, err := w.writeTask(ctx, a.Base(), tsk.Rack, existing, cfg)
	if err != nil {
		return nil, err
	}
	rec.Task = written.Key
	if err = w.writeRecord(ctx, tsk, rec, hashes); err != nil {
		return nil, err
	}
	return written, nil
}

// current returns b, or builds the Arc with key in w.tx when b is zero. When a lookup
// that b made gives another answer in w.tx, current schedules a resync of the Arc.
func (w Writer) current(ctx context.Context, b Build, key Key) (Build, error) {
	if b.lookups == nil {
		return w.rebuild(ctx, w.tx, &channel.Lookups{}, key)
	}
	same, err := b.lookups.Unchanged(ctx, w.tx)
	if err != nil {
		return Build{}, err
	}
	// A compile here would hold the write lock. A channel change schedules no resync
	// of its own.
	if !same {
		w.resyncs.Add(w.tx, key)
	}
	return b, nil
}

// hash returns the hash of a's compiled program in b and the hashes of its content. The
// program holds the code of every module a imports. An automation that does not compile
// hashes its text and the modules in previous, rehashed.
func (w Writer) hash(
	ctx context.Context,
	a Arc,
	previous drift.Hashes,
	b Build,
) (string, drift.Hashes, error) {
	text, err := Hash(a.Base())
	if err != nil {
		return "", drift.Hashes{}, err
	}
	if _, ok := a.Variant.(AutomationArc); !ok {
		return text, drift.Hashes{Text: text, Modules: previous.Modules}, nil
	}
	paths := slices.Collect(maps.Keys(previous.Modules))
	if b.err == nil {
		paths = b.loaded
	}
	modules, err := w.hashModules(ctx, w.tx, paths)
	if err != nil {
		return "", drift.Hashes{}, err
	}
	hashes := drift.Hashes{Text: text, Modules: modules}
	var hash string
	if b.err == nil {
		hash, err = HashProgram(b.program)
	} else {
		hash, err = HashContent(hashes)
	}
	if err != nil {
		return "", drift.Hashes{}, err
	}
	return hash, hashes, nil
}

// sync rewrites an automation's task config or a module's compile status from b, a
// build of its content.
func (w Writer) sync(ctx context.Context, a Arc, b Build) error {
	if module, ok := a.Variant.(ModuleArc); ok {
		return w.syncStatus(ctx, module, b)
	}
	return w.syncTask(ctx, a, b)
}

// syncStatus stores the result of b, the module's analysis, as its status.
func (w Writer) syncStatus(ctx context.Context, module ModuleArc, b Build) error {
	stat := Status{
		Owner:   new(ResourceID(module.Key)),
		Name:    module.Name,
		Variant: status.VariantSuccess,
		Message: "Compiled",
		Time:    w.now(),
	}
	if b.err != nil {
		stat.Variant = status.VariantError
		stat.Message = "Failed to compile"
		stat.Description = b.err.Error()
	}
	return w.status.NewWriter(w.tx).Set(ctx, &stat)
}

// writeTask creates the arc's task on rackKey with cfg, or applies the arc's name,
// rackKey, and cfg to the existing task, stamping the arc's key into cfg. cfg.Hash is
// the arc's compiled hash, so the task's config hash tracks the arc's content and the
// task drift mechanism reports arc content drift with no extra machinery.
func (w Writer) writeTask(
	ctx context.Context,
	a Base,
	rackKey rack.Key,
	existing []task.Key,
	cfg taskversions.Config,
) (*task.Task, error) {
	cfg.ArcKey = a.Key
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var encoded msgpack.EncodedJSON
	if err = json.Unmarshal(b, &encoded); err != nil {
		return nil, err
	}
	tw := w.tasks.NewWriter(w.tx)
	if len(existing) == 0 {
		tsk := task.Task{Rack: rackKey, Name: a.Name, Type: TaskType, Config: encoded}
		if err = tw.Create(ctx, &tsk); err != nil {
			return nil, err
		}
		return &tsk, nil
	}
	if _, err = tw.Apply(
		ctx,
		existing[0],
		task.NewSetAction(task.SetPayload{Path: []string{"name"}, Value: a.Name}),
		task.NewSetAction(task.SetPayload{Path: []string{"rack"}, Value: rackKey}),
		task.NewSetAction(task.SetPayload{Path: []string{"config"}, Value: encoded}),
	); err != nil {
		return nil, err
	}
	tsk, _, err := w.retrieveTask(ctx, existing[0])
	if err != nil {
		return nil, err
	}
	return &tsk, nil
}

// retrieveTask returns the task with the given key and its decoded config.
func (w Writer) retrieveTask(
	ctx context.Context,
	key task.Key,
) (task.Task, taskversions.Config, error) {
	var (
		tsk task.Task
		cfg taskversions.Config
	)
	if err := w.tasks.NewRetrieve().
		Where(task.MatchKeys(key)).
		Entry(&tsk).
		Exec(ctx, w.tx); err != nil {
		return tsk, cfg, err
	}
	return tsk, cfg, tsk.Config.Unmarshal(&cfg)
}

// syncTask applies the arc's name and compiled hash to the arc's task, and rewrites its
// drift record, when an edit changed either. A no-op for arcs with no rack bound and
// for edits, like graph layout moves, that change neither.
func (w Writer) syncTask(ctx context.Context, a Arc, built Build) error {
	b := a.Base()
	existing, err := retrieveTaskKeys(
		ctx, w.tasks, w.tx, w.matchTasks(b.Key),
	)
	if err != nil || len(existing) == 0 {
		return err
	}
	tsk, cfg, err := w.retrieveTask(ctx, existing[0])
	if err != nil {
		return err
	}
	rec, err := w.record(ctx, tsk.Key)
	if err != nil {
		return err
	}
	hash, hashes, err := w.hash(ctx, a, rec.Stored, built)
	if err != nil || (cfg.Hash == hash && tsk.Name == b.Name) {
		return err
	}
	cfg.Hash = hash
	if _, err = w.writeTask(ctx, b, tsk.Rack, existing, cfg); err != nil {
		return err
	}
	return w.writeRecord(ctx, tsk, rec, hashes)
}

func (w Writer) clearRack(ctx context.Context, taskKeys []task.Key) error {
	for _, taskKey := range taskKeys {
		stat, ok, err := taskStatus(ctx, w.status, w.tx, taskKey)
		if err != nil {
			return err
		}
		if ok && stat.Details.Running {
			return errors.Wrap(
				validate.ErrValidation,
				"cannot clear the rack of a running arc; stop it first",
			)
		}
		if err = w.tasks.NewWriter(w.tx).Delete(ctx, taskKey, false); err != nil {
			return err
		}
	}
	return nil
}

// taskStatus returns the status in snap of the task with key, false when none.
func taskStatus(
	ctx context.Context,
	statuses *status.Service,
	snap gorp.Snapshot,
	key task.Key,
) (task.Status, bool, error) {
	var stat task.Status
	err := statuses.NewRetrieve[task.StatusDetails]().
		Where(status.MatchOwners[task.StatusDetails](task.ResourceID(key))).
		Entry(&stat).
		Exec(ctx, snap)
	if errors.Is(err, query.ErrNotFound) {
		return stat, false, nil
	}
	return stat, err == nil, err
}
