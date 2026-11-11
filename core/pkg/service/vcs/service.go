// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package vcs keeps the configuration of the Core in a Git repository. It exports the
// configuration as files, commits and pushes it, and applies commits back to the live
// configuration in one transaction.
package vcs

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for opening a Service.
type Config struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
	// DB holds the live configuration.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Dirname is the data directory of the Core. The repository sits in its vcs folder.
	//
	// [OPTIONAL] - Defaults to a temporary directory that Close removes.
	Dirname string
	// Channel reads and writes channels.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Task reads and writes tasks.
	//
	// [REQUIRED]
	Task *task.Service
	// TaskStopper stops a task before a delete and restarts a changed task.
	//
	// [REQUIRED]
	TaskStopper *task.Stopper
	// Status reports which tasks run.
	//
	// [REQUIRED]
	Status *status.Service
	// Arc reads and writes Arcs and their directories.
	//
	// [REQUIRED]
	Arc *arc.Service
	// MatchArcTasks returns a filter for the tasks that run the given Arcs.
	//
	// [REQUIRED]
	MatchArcTasks func(arcs ...arc.Key) task.Filter
	// Schematic reads and writes schematics and schematic symbols.
	//
	// [REQUIRED]
	Schematic *schematic.Service
	// LinePlot reads and writes line plots.
	//
	// [REQUIRED]
	LinePlot *lineplot.Service
	// Log reads and writes logs.
	//
	// [REQUIRED]
	Log *log.Service
	// Table reads and writes tables.
	//
	// [REQUIRED]
	Table *table.Service
	// Panel reads and writes panels.
	//
	// [REQUIRED]
	Panel *panel.Service
	// Project reads and writes projects.
	//
	// [REQUIRED]
	Project *project.Service
	// Group reads and writes groups.
	//
	// [REQUIRED]
	Group *group.Service
	// CoreGroups are the channel groups that the Core makes for itself, such as host
	// metrics. Their channels are not part of a configuration.
	//
	// [OPTIONAL]
	CoreGroups []group.Key
	// FetchInterval is how often the Service fetches the remote.
	//
	// [OPTIONAL] - Defaults to 30 seconds.
	FetchInterval time.Duration
	// StatusDebounce is how long the Service waits after a change to the live
	// configuration before it computes the status again.
	//
	// [OPTIONAL] - Defaults to 300 milliseconds.
	StatusDebounce time.Duration
	// RestartTimeout caps how long an apply waits for an Arc task to take a new config
	// before it restarts the task.
	//
	// [OPTIONAL] - Defaults to 5 seconds.
	RestartTimeout time.Duration
	// HTTPClient sends the requests to the GitHub API.
	//
	// [OPTIONAL] - Defaults to http.DefaultClient.
	HTTPClient *http.Client
	// GitHubAPI is the base URL of the GitHub API.
	//
	// [OPTIONAL] - Defaults to https://api.github.com.
	GitHubAPI string
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a Service.
	DefaultConfig = Config{
		FetchInterval:  30 * time.Second,
		StatusDebounce: 300 * time.Millisecond,
		RestartTimeout: 5 * time.Second,
		HTTPClient:     http.DefaultClient,
		GitHubAPI:      "https://api.github.com",
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Dirname = override.String(c.Dirname, other.Dirname)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Task = override.Nil(c.Task, other.Task)
	c.TaskStopper = override.Nil(c.TaskStopper, other.TaskStopper)
	c.Status = override.Nil(c.Status, other.Status)
	c.Arc = override.Nil(c.Arc, other.Arc)
	c.MatchArcTasks = override.Nil(c.MatchArcTasks, other.MatchArcTasks)
	c.Schematic = override.Nil(c.Schematic, other.Schematic)
	c.LinePlot = override.Nil(c.LinePlot, other.LinePlot)
	c.Log = override.Nil(c.Log, other.Log)
	c.Table = override.Nil(c.Table, other.Table)
	c.Panel = override.Nil(c.Panel, other.Panel)
	c.Project = override.Nil(c.Project, other.Project)
	c.Group = override.Nil(c.Group, other.Group)
	c.CoreGroups = override.Slice(c.CoreGroups, other.CoreGroups)
	c.FetchInterval = override.Numeric(c.FetchInterval, other.FetchInterval)
	c.StatusDebounce = override.Numeric(c.StatusDebounce, other.StatusDebounce)
	c.RestartTimeout = override.Numeric(c.RestartTimeout, other.RestartTimeout)
	c.HTTPClient = override.Nil(c.HTTPClient, other.HTTPClient)
	c.GitHubAPI = override.String(c.GitHubAPI, other.GitHubAPI)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("vcs")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	v.NotNil("task", c.Task)
	v.NotNil("task_stopper", c.TaskStopper)
	v.NotNil("status", c.Status)
	v.NotNil("arc", c.Arc)
	v.NotNil("match_arc_tasks", c.MatchArcTasks)
	v.NotNil("schematic", c.Schematic)
	v.NotNil("lineplot", c.LinePlot)
	v.NotNil("log", c.Log)
	v.NotNil("table", c.Table)
	v.NotNil("panel", c.Panel)
	v.NotNil("project", c.Project)
	v.NotNil("group", c.Group)
	v.Positive("fetch_interval", c.FetchInterval)
	v.Positive("status_debounce", c.StatusDebounce)
	v.Positive("restart_timeout", c.RestartTimeout)
	v.NotNil("http_client", c.HTTPClient)
	v.NotEmptyString("github_api", c.GitHubAPI)
	return v.Error()
}

// Write runs apply in one write transaction and commits it. The API passes a Write that
// checks access on what the transaction changed.
type Write func(ctx context.Context, apply func(context.Context, gorp.Tx) error) error

// Service keeps the configuration of the Core in a Git repository. It is safe for
// concurrent use: it runs one operation at a time.
type Service struct {
	cfg Config
	dir string
	// temp is true when the Service made dir and removes it on Close.
	temp bool
	// mu guards repo, conn, fetchedAt, and lastErr.
	mu        sync.Mutex
	repo      *repo
	conn      connection
	fetchedAt telem.TimeStamp
	lastErr   string
	// statusMu guards status, so Status does not wait for a running fetch.
	statusMu sync.RWMutex
	status   *Status
	changed  chan struct{}
	stop     observe.Disconnect
	shutdown io.Closer
}

// OpenService opens the repository in the data directory and starts the status and
// fetch loops.
func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, changed: make(chan struct{}, 1)}
	if cfg.Dirname == "" {
		if s.dir, err = os.MkdirTemp("", "synnax-vcs-*"); err != nil {
			return nil, err
		}
		s.temp = true
	} else {
		s.dir = filepath.Join(cfg.Dirname, "vcs")
	}
	if err = os.MkdirAll(s.dir, 0o700); err != nil {
		return nil, err
	}
	if s.repo, err = openRepo(filepath.Join(s.dir, "repo")); err != nil {
		return nil, err
	}
	if s.conn, err = readConnection(s.dir); err != nil {
		return nil, err
	}
	if err = s.repo.setRemote(s.conn.RemoteURL); err != nil {
		return nil, err
	}
	if s.enabled() {
		if _, err = s.current(ctx); err != nil {
			return nil, err
		}
	}
	s.stop = cfg.DB.Observe().OnChange(s.onCommit)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	s.shutdown = signal.NewHardShutdown(sCtx, cancel)
	sCtx.Go(s.watch, signal.WithKey("vcs.status"), signal.RecoverWithErrOnPanic())
	signal.GoTick(
		sCtx,
		cfg.FetchInterval,
		func(ctx context.Context, _ time.Time) error {
			s.fetchInBackground(ctx)
			return nil
		},
		signal.WithKey("vcs.fetch"),
		signal.WithRetryOnPanic(),
	)
	return s, nil
}

// Close stops the loops of the Service.
func (s *Service) Close() error {
	s.stop()
	err := s.shutdown.Close()
	if s.temp {
		err = errors.Combine(err, os.RemoveAll(s.dir))
	}
	return err
}

// onCommit runs on the committing goroutine, so it only signals the status loop.
func (s *Service) onCommit(_ context.Context, c gorp.Commit) {
	for _, ch := range c.Changes {
		if versionedTable(ch.Record.Table) {
			select {
			case s.changed <- struct{}{}:
			default:
			}
			return
		}
	}
}

// versionedTable reports whether a change to table can change the configuration.
// Statuses change all the time and are never part of it.
func versionedTable(table string) bool {
	return !strings.Contains(strings.ToLower(table), "status")
}

func (s *Service) watch(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.changed:
		}
		t := time.NewTimer(s.cfg.StatusDebounce)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
		s.mu.Lock()
		if s.enabled() {
			s.refresh(ctx)
		}
		s.mu.Unlock()
	}
}

func (s *Service) fetchInBackground(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled() || s.conn.RemoteURL == "" {
		return
	}
	s.fetch(ctx)
	s.refresh(ctx)
}

func (s *Service) enabled() bool { return s.conn.Configuration != "" }

func (s *Service) requireEnabled() error {
	if !s.enabled() {
		return ErrNotStarted
	}
	return nil
}

// fetch fetches the remote and records the outcome in the status.
func (s *Service) fetch(ctx context.Context) {
	if err := s.repo.fetch(ctx, s.conn.remote()); err != nil {
		s.fail("fetch", err)
		return
	}
	s.fetchedAt = telem.Now()
	s.lastErr = ""
}

// push pushes the tracked configuration and records a failure in the status. A failed
// push leaves the commits waiting, so it is no error for the caller.
func (s *Service) push(ctx context.Context) {
	if s.conn.RemoteURL == "" {
		return
	}
	if err := s.repo.push(ctx, s.conn.remote(), s.conn.Configuration); err != nil {
		s.fail("push", err)
		return
	}
	s.lastErr = ""
}

func (s *Service) fail(op string, err error) {
	s.lastErr = op + ": " + err.Error()
	s.cfg.L.Warn("version control "+op+" failed", zap.Error(err))
}

// refresh computes the status again and logs a failure, since its callers have no one
// to return it to.
func (s *Service) refresh(ctx context.Context) {
	st, err := s.compute(ctx)
	if err != nil {
		s.cache(nil)
		s.cfg.L.Error("failed to compute version control status", zap.Error(err))
		return
	}
	s.cache(&st)
}

func (s *Service) cache(st *Status) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	s.status = st
}

func (s *Service) cached() (Status, bool) {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	if s.status == nil {
		return Status{}, false
	}
	return *s.status, true
}

// Status returns the state of version control. It returns the cached state at once,
// also while a fetch or another operation runs.
func (s *Service) Status(ctx context.Context) (Status, error) {
	if st, ok := s.cached(); ok {
		return st, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current(ctx)
}

func (s *Service) current(ctx context.Context) (Status, error) {
	if st, ok := s.cached(); ok {
		return st, nil
	}
	st, err := s.compute(ctx)
	if err != nil {
		return Status{}, err
	}
	s.cache(&st)
	return st, nil
}

func (s *Service) compute(ctx context.Context) (Status, error) {
	st := Status{
		Enabled:         s.enabled(),
		Configuration:   s.conn.Configuration,
		RemoteURL:       s.conn.RemoteURL,
		TokenSet:        s.conn.Token != "",
		Changes:         []Change{},
		Incoming:        []Commit{},
		IncomingChanges: []Change{},
		FetchedAt:       int64(s.fetchedAt),
		Error:           s.lastErr,
	}
	if !st.Enabled {
		return st, nil
	}
	head, headTree, err := s.head()
	if err != nil {
		return Status{}, err
	}
	if !head.IsZero() {
		c, err := s.repo.toCommit(head)
		if err != nil {
			return Status{}, err
		}
		st.Head = &c
	}
	work, err := s.working(ctx)
	if err != nil {
		return Status{}, err
	}
	st.Changes = work.diff(headTree)
	s.markReviews(st.Changes)
	if s.conn.RemoteURL == "" {
		return st, nil
	}
	origin, _, err := s.repo.ref(remoteRef(s.conn.Configuration))
	if err != nil {
		return Status{}, err
	}
	incoming, err := s.repo.between(origin, head)
	if err != nil {
		return Status{}, err
	}
	for _, c := range incoming {
		st.Incoming = append(st.Incoming, wireCommit(c))
	}
	if len(incoming) > 0 {
		base, err := s.repo.mergeBase(head, origin)
		if err != nil {
			return Status{}, err
		}
		if st.IncomingChanges, err = s.diffCommits(base, origin); err != nil {
			return Status{}, err
		}
	}
	waiting, err := s.repo.between(head, origin)
	if err != nil {
		return Status{}, err
	}
	st.Waiting = len(waiting)
	return st, nil
}

// markReviews sets the open pull request of each change, and forgets a pull request
// that holds no change.
func (s *Service) markReviews(changes []Change) {
	byResource := map[string]int{}
	for i, c := range changes {
		byResource[c.Resource] = i
	}
	kept := s.conn.Reviews[:0]
	for _, r := range s.conn.Reviews {
		held := false
		for _, res := range r.Resources {
			if i, ok := byResource[res]; ok {
				changes[i].Review = &Review{Number: r.Number, URL: r.URL}
				held = true
			}
		}
		if held {
			kept = append(kept, r)
		}
	}
	if len(kept) == len(s.conn.Reviews) {
		return
	}
	s.conn.Reviews = kept
	if err := writeConnection(s.dir, s.conn); err != nil {
		s.cfg.L.Warn("failed to store version control connection", zap.Error(err))
	}
}

// head returns the tip of the tracked configuration and its tree. The hash is zero when
// the configuration has no local commit.
func (s *Service) head() (head plumbing.Hash, t *tree, err error) {
	if head, _, err = s.repo.ref(branchRef(s.conn.Configuration)); err != nil {
		return head, nil, err
	}
	t, err = s.repo.readTree(head)
	return head, t, err
}

// diffCommits returns the resources that differ from the commit from to the commit to.
func (s *Service) diffCommits(from, to plumbing.Hash) ([]Change, error) {
	a, err := s.repo.readTree(from)
	if err != nil {
		return nil, err
	}
	b, err := s.repo.readTree(to)
	if err != nil {
		return nil, err
	}
	return b.diff(a), nil
}

// working exports the live configuration with the generated files of the repository.
func (s *Service) working(ctx context.Context) (*tree, error) {
	var t *tree
	if err := s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
		t, err = s.export(ctx, snap)
		return err
	}); err != nil {
		return nil, err
	}
	t.raw[readmePath] = []byte(readme)
	t.raw[manifestPath] = []byte(manifest)
	return t, nil
}

const manifest = `{
  "format": 1
}
`

const readme = `# Synnax configuration

This repository holds the configuration of a Synnax Core. A commit in the Console
writes these files. You can also edit them here: the Console shows new commits as
incoming changes, and a pull applies them to the Core.

- channels: one file per channel.
- tasks: one file per task, with its config and the rack it runs on.
- arc: one folder per Arc directory. An Arc has a .json file and, for a text Arc, a
  .arc file with its source.
- symbols: one file per schematic symbol, in a folder per group.
- groups: one folder per top-level project group.
- projects: one folder per project. A file named <name>.<type>.json holds a
  schematic, line plot, log, table, or panel.
`
