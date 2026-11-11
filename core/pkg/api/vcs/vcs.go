// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package vcs exposes version control of the configuration of the Core over the API.
package vcs

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/vcs"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the version control API service.
type ServiceConfig struct {
	// DB opens the snapshots that access checks read.
	//
	// [REQUIRED]
	DB *gorp.DB
	// RBAC checks access on version control.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Binder checks access on what an apply, a discard, or a write changes.
	//
	// [REQUIRED]
	Binder *bind.Service
	// VCS is the version control service.
	//
	// [REQUIRED]
	VCS *vcs.Service
	// User names the author of a commit.
	//
	// [REQUIRED]
	User *user.Service
	// Password holds the username of the author of a commit.
	//
	// [REQUIRED]
	Password *password.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the version control API
	// service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Binder = override.Nil(c.Binder, other.Binder)
	c.VCS = override.Nil(c.VCS, other.VCS)
	c.User = override.Nil(c.User, other.User)
	c.Password = override.Nil(c.Password, other.Password)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.vcs")
	v.NotNil("db", c.DB)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("binder", c.Binder)
	v.NotNil("vcs", c.VCS)
	v.NotNil("user", c.User)
	v.NotNil("password", c.Password)
	return v.Error()
}

// Service is the version control API service. Each route checks one action on the
// version control builtin before it runs. Version control holds a lock across network
// calls, so no route runs inside a snapshot or a transaction of its own.
type Service struct{ cfg ServiceConfig }

// NewService creates a Service from the given configurations.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

func (s *Service) enforce(ctx context.Context, action access.Action) error {
	return s.enforceOn(ctx, action, []resource.ID{vcs.ResourceID})
}

func (s *Service) enforceOn(
	ctx context.Context,
	action access.Action,
	objects []resource.ID,
) error {
	if len(objects) == 0 {
		return nil
	}
	return s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.cfg.RBAC.NewEnforcer(snap).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  action,
			Objects: objects,
		})
	})
}

// enforceRetrieve checks that the subject can retrieve each resource of ids, or of
// every uncommitted change when ids is empty. A commit or snapshot sends their files
// to the remote.
func (s *Service) enforceRetrieve(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		st, err := s.cfg.VCS.Status(ctx)
		if err != nil {
			return err
		}
		for _, c := range st.Changes {
			ids = append(ids, c.Resource)
		}
	}
	objects := make([]resource.ID, 0, len(ids))
	for _, r := range ids {
		id, err := resource.ParseID(r)
		if err != nil {
			return err
		}
		objects = append(objects, id)
	}
	return s.enforceOn(ctx, access.ActionRetrieve, objects)
}

// write runs apply in a transaction that checks access on each change, as bind.Write
// does.
func (s *Service) write(
	ctx context.Context,
	apply func(context.Context, gorp.Tx) error,
) error {
	_, err := bind.Write(s.cfg.Binder, func(
		ctx context.Context,
		tx gorp.Tx,
		_ struct{},
	) (struct{}, error) {
		return struct{}{}, apply(ctx, tx)
	})(ctx, struct{}{})
	return err
}

// author returns the person that commits for the subject of ctx: the display name of
// the user, and an email address made from the username.
func (s *Service) author(ctx context.Context) (vcs.Author, error) {
	subject := auth.GetSubject(ctx)
	name, username := "", subject.Key
	var p password.Password
	err := s.cfg.Password.NewRetrieve().
		Where(password.MatchSubjects(subject)).
		Entry(&p).
		Exec(ctx, s.cfg.DB)
	if err == nil {
		username = p.Username
	} else if !errors.Is(err, query.ErrNotFound) {
		return vcs.Author{}, err
	}
	if subject.Type == resource.TypeUser {
		key, err := resource.ParseUUIDKey(subject)
		if err != nil {
			return vcs.Author{}, err
		}
		var u user.User
		err = s.cfg.User.NewRetrieve().
			Where(user.MatchKeys(key)).
			Entry(&u).
			Exec(ctx, s.cfg.DB)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
			return vcs.Author{}, err
		}
		name = u.Name
	}
	if name == "" {
		name = username
	}
	return vcs.Author{Name: name, Email: username + "@synnax.local"}, nil
}

type (
	// StatusRequest is a request for the state of version control.
	StatusRequest struct{}
	// ConnectRequest starts version control or changes its configuration and remote.
	ConnectRequest struct {
		Configuration string `json:"configuration" msgpack:"configuration"`
		RemoteURL     string `json:"remote_url"    msgpack:"remote_url"`
		// Token is never returned. Empty keeps the stored token.
		Token string `json:"token" msgpack:"token"`
	}
	// FetchRequest fetches the remote now.
	FetchRequest struct{}
	// CommitRequest commits the uncommitted changes of the selected resources.
	CommitRequest struct {
		Message     string   `json:"message"      msgpack:"message"`
		Resources   []string `json:"resources"    msgpack:"resources"`
		PullRequest bool     `json:"pull_request" msgpack:"pull_request"`
	}
	// CommitResponse is the new commit and the pull request that holds it, if any.
	CommitResponse struct {
		Commit         vcs.Commit `json:"commit"           msgpack:"commit"`
		PullRequestURL string     `json:"pull_request_url" msgpack:"pull_request_url"`
	}
	// DiscardRequest puts the selected resources back as the head holds them.
	DiscardRequest struct {
		Resources []string `json:"resources" msgpack:"resources"`
	}
	// PlanRequest asks what applying a target changes.
	PlanRequest struct {
		Target vcs.Target `json:"target" msgpack:"target"`
	}
	// ApplyRequest applies a target with a choice for each conflict.
	ApplyRequest struct {
		Target  vcs.Target   `json:"target"  msgpack:"target"`
		Choices []vcs.Choice `json:"choices" msgpack:"choices"`
	}
	// FilesRequest asks for the files of a resource in a revision.
	FilesRequest struct {
		Rev      string `json:"rev"      msgpack:"rev"`
		Resource string `json:"resource" msgpack:"resource"`
	}
	// FilesResponse holds the files, sorted by path.
	FilesResponse struct {
		Files []vcs.File `json:"files" msgpack:"files"`
	}
	// LogRequest asks for the commits of a configuration.
	LogRequest struct {
		Resource      string `json:"resource"      msgpack:"resource"`
		Configuration string `json:"configuration" msgpack:"configuration"`
		Limit         int    `json:"limit"         msgpack:"limit"`
	}
	// LogResponse holds the commits, newest first.
	LogResponse struct {
		Commits []vcs.Commit `json:"commits" msgpack:"commits"`
	}
	// ShowRequest asks for one commit and its changes.
	ShowRequest struct {
		Commit string `json:"commit" msgpack:"commit"`
	}
	// ShowResponse holds the commit with its changes.
	ShowResponse struct {
		Commit vcs.Commit `json:"commit" msgpack:"commit"`
	}
	// ConfigurationsRequest asks for every configuration.
	ConfigurationsRequest struct{}
	// ConfigurationsResponse holds the configurations, sorted by name.
	ConfigurationsResponse struct {
		Configurations []vcs.Configuration `json:"configurations" msgpack:"configurations"`
	}
	// CloneRequest starts a configuration from the tracked one.
	CloneRequest struct {
		Name string `json:"name" msgpack:"name"`
	}
	// CompareRequest asks for the resources that differ between two revisions.
	CompareRequest struct {
		From string `json:"from" msgpack:"from"`
		To   string `json:"to"   msgpack:"to"`
	}
	// CompareResponse holds the changes from From to To.
	CompareResponse struct {
		Changes []vcs.Change `json:"changes" msgpack:"changes"`
	}
	// SnapshotRequest tags the live configuration for a range.
	SnapshotRequest struct {
		Range string `json:"range" msgpack:"range"`
	}
	// SnapshotResponse holds the new snapshot.
	SnapshotResponse struct {
		Snapshot vcs.Snapshot `json:"snapshot" msgpack:"snapshot"`
	}
	// SnapshotsRequest asks for the snapshots of a range.
	SnapshotsRequest struct {
		Range string `json:"range" msgpack:"range"`
	}
	// SnapshotsResponse holds the snapshots, newest first.
	SnapshotsResponse struct {
		Snapshots []vcs.Snapshot `json:"snapshots" msgpack:"snapshots"`
	}
)

// Status returns the state of version control. It needs retrieve.
func (s *Service) Status(ctx context.Context, _ StatusRequest) (vcs.Status, error) {
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return vcs.Status{}, err
	}
	return s.cfg.VCS.Status(ctx)
}

// Connect starts version control or changes its configuration and remote. It stores a
// token, so it needs delete, which only the Owner holds.
func (s *Service) Connect(ctx context.Context, req ConnectRequest) (vcs.Status, error) {
	if err := s.enforce(ctx, access.ActionDelete); err != nil {
		return vcs.Status{}, err
	}
	author, err := s.author(ctx)
	if err != nil {
		return vcs.Status{}, err
	}
	return s.cfg.VCS.Connect(ctx, vcs.ConnectRequest{
		Configuration: req.Configuration,
		RemoteURL:     req.RemoteURL,
		Token:         req.Token,
		Author:        author,
	})
}

// Fetch fetches the remote now. It needs retrieve.
func (s *Service) Fetch(ctx context.Context, _ FetchRequest) (vcs.Status, error) {
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return vcs.Status{}, err
	}
	return s.cfg.VCS.Fetch(ctx)
}

// Commit commits the selected changes. It needs update, and retrieve on each resource
// it commits.
func (s *Service) Commit(
	ctx context.Context,
	req CommitRequest,
) (CommitResponse, error) {
	if err := s.enforce(ctx, access.ActionUpdate); err != nil {
		return CommitResponse{}, err
	}
	if err := s.enforceRetrieve(ctx, req.Resources); err != nil {
		return CommitResponse{}, err
	}
	author, err := s.author(ctx)
	if err != nil {
		return CommitResponse{}, err
	}
	c, url, err := s.cfg.VCS.Commit(ctx, vcs.CommitRequest{
		Message:     req.Message,
		Resources:   req.Resources,
		PullRequest: req.PullRequest,
		Author:      author,
	})
	if err != nil {
		return CommitResponse{}, err
	}
	return CommitResponse{Commit: c, PullRequestURL: url}, nil
}

// Discard puts the selected resources back as the head holds them. It needs update, and
// access to each resource it changes.
func (s *Service) Discard(
	ctx context.Context,
	req DiscardRequest,
) (vcs.Inverse, error) {
	if err := s.enforce(ctx, access.ActionUpdate); err != nil {
		return vcs.Inverse{}, err
	}
	return s.cfg.VCS.Discard(ctx, req.Resources, s.write)
}

// Write undoes a discard or an apply. It needs update, and access to each resource it
// changes.
func (s *Service) Write(ctx context.Context, req vcs.Inverse) (struct{}, error) {
	if err := s.enforce(ctx, access.ActionUpdate); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.cfg.VCS.Write(ctx, req, s.write)
}

// Plan returns what applying a target changes. It needs retrieve.
func (s *Service) Plan(ctx context.Context, req PlanRequest) (vcs.Plan, error) {
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return vcs.Plan{}, err
	}
	return s.cfg.VCS.Plan(ctx, req.Target)
}

// Apply applies a target. It needs update, and access to each resource it changes.
func (s *Service) Apply(ctx context.Context, req ApplyRequest) (vcs.Inverse, error) {
	if err := s.enforce(ctx, access.ActionUpdate); err != nil {
		return vcs.Inverse{}, err
	}
	return s.cfg.VCS.Apply(ctx, req.Target, req.Choices, s.write)
}

// Files returns the files of a resource in a revision. It needs retrieve, on the
// resource too.
func (s *Service) Files(ctx context.Context, req FilesRequest) (FilesResponse, error) {
	if req.Resource == "" {
		return FilesResponse{}, errors.Wrap(
			validate.ErrValidation,
			"files: a resource is required",
		)
	}
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return FilesResponse{}, err
	}
	if err := s.enforceRetrieve(ctx, []string{req.Resource}); err != nil {
		return FilesResponse{}, err
	}
	fs, err := s.cfg.VCS.Files(ctx, req.Rev, req.Resource)
	return FilesResponse{Files: fs}, err
}

// Log returns the commits of a configuration. It needs retrieve.
func (s *Service) Log(ctx context.Context, req LogRequest) (LogResponse, error) {
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return LogResponse{}, err
	}
	commits, err := s.cfg.VCS.Log(ctx, vcs.LogRequest{
		Resource:      req.Resource,
		Configuration: req.Configuration,
		Limit:         req.Limit,
	})
	return LogResponse{Commits: commits}, err
}

// Show returns one commit and its changes. It needs retrieve.
func (s *Service) Show(ctx context.Context, req ShowRequest) (ShowResponse, error) {
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return ShowResponse{}, err
	}
	c, err := s.cfg.VCS.Show(ctx, req.Commit)
	return ShowResponse{Commit: c}, err
}

// Configurations returns every configuration. It needs retrieve.
func (s *Service) Configurations(
	ctx context.Context,
	_ ConfigurationsRequest,
) (ConfigurationsResponse, error) {
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return ConfigurationsResponse{}, err
	}
	confs, err := s.cfg.VCS.Configurations(ctx)
	return ConfigurationsResponse{Configurations: confs}, err
}

// Clone starts a configuration from the tracked one. It needs update.
func (s *Service) Clone(ctx context.Context, req CloneRequest) (vcs.Status, error) {
	if err := s.enforce(ctx, access.ActionUpdate); err != nil {
		return vcs.Status{}, err
	}
	return s.cfg.VCS.Clone(ctx, req.Name)
}

// Compare returns the resources that differ between two revisions. It needs retrieve.
func (s *Service) Compare(
	ctx context.Context,
	req CompareRequest,
) (CompareResponse, error) {
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return CompareResponse{}, err
	}
	changes, err := s.cfg.VCS.Compare(ctx, req.From, req.To)
	return CompareResponse{Changes: changes}, err
}

// Snapshot tags the live configuration for a range. It needs create, and retrieve on
// each uncommitted change.
func (s *Service) Snapshot(
	ctx context.Context,
	req SnapshotRequest,
) (SnapshotResponse, error) {
	if err := s.enforce(ctx, access.ActionCreate); err != nil {
		return SnapshotResponse{}, err
	}
	if err := s.enforceRetrieve(ctx, nil); err != nil {
		return SnapshotResponse{}, err
	}
	author, err := s.author(ctx)
	if err != nil {
		return SnapshotResponse{}, err
	}
	snap, err := s.cfg.VCS.Snapshot(ctx, req.Range, author)
	return SnapshotResponse{Snapshot: snap}, err
}

// Snapshots returns the snapshots of a range. It needs retrieve.
func (s *Service) Snapshots(
	ctx context.Context,
	req SnapshotsRequest,
) (SnapshotsResponse, error) {
	if err := s.enforce(ctx, access.ActionRetrieve); err != nil {
		return SnapshotsResponse{}, err
	}
	snaps, err := s.cfg.VCS.Snapshots(ctx, req.Range)
	return SnapshotsResponse{Snapshots: snaps}, err
}
