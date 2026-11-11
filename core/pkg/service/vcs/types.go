// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package vcs

import (
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ResourceKey is the key of the builtin resource that access checks on version control
// name.
const ResourceKey = "version_control"

// ResourceID is the builtin resource that access checks on version control name.
var ResourceID = resource.ID{Type: resource.TypeBuiltin, Key: ResourceKey}

const (
	// RevWorking names the live configuration of the Core.
	RevWorking = "working"
	// RevHead names the commit that the live configuration is based on.
	RevHead = "head"
	// revConfigurationPrefix starts a revision that names the tip of a configuration.
	revConfigurationPrefix = "configuration/"
)

var (
	// ErrNotStarted is returned by an operation that needs version control started.
	ErrNotStarted = errors.Wrap(
		validate.ErrValidation,
		"version control is not started",
	)
	// ErrUnresolvedConflict is returned by Apply when a conflict has no choice.
	ErrUnresolvedConflict = errors.Wrap(
		validate.ErrValidation,
		"every conflict needs a choice",
	)
	// ErrMoved is returned by Apply when the commit of a pull or load target is no
	// longer the commit the plan reviewed.
	ErrMoved = errors.Mark(
		errors.New(
			"The commits to apply changed after you reviewed them. Open the plan again.",
		),
		validate.ErrValidation,
	)
)

// File is one file of a configuration in the repository.
type File struct {
	// Path is the slash-separated path from the repository root.
	Path string `json:"path" msgpack:"path"`
	// Content is the content of the file.
	Content string `json:"content" msgpack:"content"`
}

// Inverse undoes a change to the live configuration: Write puts Files back and deletes
// the resources in Removed.
type Inverse struct {
	// Files are the files of the resources to put back.
	Files []File `json:"files" msgpack:"files"`
	// Removed holds the IDs of the resources to delete.
	Removed []string `json:"removed" msgpack:"removed"`
	// Configuration is the configuration to track again, or empty to keep the tracked
	// one. It undoes a load.
	Configuration string `json:"configuration" msgpack:"configuration"`
}

// ChangeKind is how a resource differs between two revisions.
type ChangeKind string

const (
	// ChangeAdded is a resource that only the newer revision holds.
	ChangeAdded ChangeKind = "added"
	// ChangeModified is a resource whose files differ between the revisions.
	ChangeModified ChangeKind = "modified"
	// ChangeDeleted is a resource that only the older revision holds.
	ChangeDeleted ChangeKind = "deleted"
)

// Review is an open pull request that holds a change.
type Review struct {
	// Number is the number of the pull request.
	Number int `json:"number" msgpack:"number"`
	// URL is the web address of the pull request.
	URL string `json:"url" msgpack:"url"`
}

// Change is one resource that differs between two revisions.
type Change struct {
	// Resource is the ID of the resource, as "<type>:<key>".
	Resource string `json:"resource" msgpack:"resource"`
	// Type is the resource type.
	Type string `json:"type" msgpack:"type"`
	// Name is the name of the resource in the newer revision, or in the older one for a
	// deletion.
	Name string `json:"name" msgpack:"name"`
	// Kind is how the resource differs.
	Kind ChangeKind `json:"kind" msgpack:"kind"`
	// Path is the path of the main file of the resource.
	Path string `json:"path" msgpack:"path"`
	// Project is the ID of the project whose folder holds the resource, as
	// "project:<key>", or empty for a resource outside every project.
	Project string `json:"project" msgpack:"project"`
	// Digest changes whenever the newer files of the resource change. It is empty for a
	// deletion.
	Digest string `json:"digest" msgpack:"digest"`
	// Review is the open pull request that holds the change, if any.
	Review *Review `json:"review,omitzero" msgpack:"review,omitempty"`
}

// Commit is a commit in the repository.
type Commit struct {
	// Hash is the full hash of the commit.
	Hash string `json:"hash" msgpack:"hash"`
	// ShortHash is the first seven characters of Hash.
	ShortHash string `json:"short_hash" msgpack:"short_hash"`
	// Message is the commit message.
	Message string `json:"message" msgpack:"message"`
	// Author is the name of the author.
	Author string `json:"author" msgpack:"author"`
	// Time is when the author made the commit, in nanoseconds since the Unix epoch.
	Time int64 `json:"time" msgpack:"time"`
	// Changes are the resources that the commit changed from its parent.
	Changes []Change `json:"changes,omitzero" msgpack:"changes,omitempty"`
}

// Status is the state of version control on the Core.
type Status struct {
	// Enabled is true once version control has started.
	Enabled bool `json:"enabled" msgpack:"enabled"`
	// Configuration is the name of the tracked configuration.
	Configuration string `json:"configuration" msgpack:"configuration"`
	// RemoteURL is the URL of the remote repository, or empty for none.
	RemoteURL string `json:"remote_url" msgpack:"remote_url"`
	// TokenSet is true when a token for the remote is stored.
	TokenSet bool `json:"token_set" msgpack:"token_set"`
	// Head is the commit that the live configuration is based on.
	Head *Commit `json:"head,omitzero" msgpack:"head,omitempty"`
	// Changes are the uncommitted changes of the live configuration.
	Changes []Change `json:"changes" msgpack:"changes"`
	// Incoming are the commits on the remote that Head does not hold.
	Incoming []Commit `json:"incoming" msgpack:"incoming"`
	// IncomingChanges are the resources that Incoming changes.
	IncomingChanges []Change `json:"incoming_changes" msgpack:"incoming_changes"`
	// Waiting is the number of commits that wait for a push.
	Waiting int `json:"waiting" msgpack:"waiting"`
	// FetchedAt is when the last fetch succeeded, in nanoseconds since the Unix epoch.
	FetchedAt int64 `json:"fetched_at" msgpack:"fetched_at"`
	// Error is the last fetch or push error, or empty when there is none.
	Error string `json:"error" msgpack:"error"`
}

// TargetKind selects what a Plan applies to the live configuration.
type TargetKind string

const (
	// TargetPull applies the commits on the remote.
	TargetPull TargetKind = "pull"
	// TargetRestore applies the files of a commit and leaves the result uncommitted.
	TargetRestore TargetKind = "restore"
	// TargetLoad applies a configuration and tracks it.
	TargetLoad TargetKind = "load"
)

// Target is what a Plan applies to the live configuration.
type Target struct {
	// Kind selects the target.
	Kind TargetKind `json:"kind" msgpack:"kind"`
	// Commit is the commit to restore. Plan sets it to the commit that a pull or load
	// moves to, and Apply then refuses the target with ErrMoved once that moves.
	Commit string `json:"commit,omitzero" msgpack:"commit,omitempty"`
	// Configuration is the configuration to load.
	Configuration string `json:"configuration,omitzero" msgpack:"configuration,omitempty"`
	// Resources limits a restore to these resource IDs. Empty restores everything.
	Resources []string `json:"resources,omitzero" msgpack:"resources,omitempty"`
}

// PlanItemKind is what applying a plan does to one resource.
type PlanItemKind string

const (
	// PlanCreate creates the resource.
	PlanCreate PlanItemKind = "create"
	// PlanChange replaces the resource.
	PlanChange PlanItemKind = "change"
	// PlanDelete deletes the resource.
	PlanDelete PlanItemKind = "delete"
)

// PlanItem is one resource that applying a plan changes.
type PlanItem struct {
	// Resource is the ID of the resource.
	Resource string `json:"resource" msgpack:"resource"`
	// Type is the resource type.
	Type string `json:"type" msgpack:"type"`
	// Name is the name of the resource.
	Name string `json:"name" msgpack:"name"`
	// Kind is what the apply does.
	Kind PlanItemKind `json:"kind" msgpack:"kind"`
	// Conflict is true when the live configuration and the target both changed the
	// resource. Apply needs a Choice for it.
	Conflict bool `json:"conflict" msgpack:"conflict"`
	// Running is true when the resource is a task or an Arc that runs.
	Running bool `json:"running" msgpack:"running"`
	// Restarts is true when the apply restarts the running task.
	Restarts bool `json:"restarts" msgpack:"restarts"`
}

// Plan is what applying a target changes on the live configuration.
type Plan struct {
	// Target is the target of the plan.
	Target Target `json:"target" msgpack:"target"`
	// Items are the resources that the apply changes.
	Items []PlanItem `json:"items" msgpack:"items"`
}

// ChoiceKind resolves a conflict.
type ChoiceKind string

const (
	// ChoiceMine keeps the live resource.
	ChoiceMine ChoiceKind = "mine"
	// ChoiceIncoming takes the resource of the target.
	ChoiceIncoming ChoiceKind = "incoming"
)

// Choice resolves the conflict on one resource.
type Choice struct {
	// Resource is the ID of the resource.
	Resource string `json:"resource" msgpack:"resource"`
	// Choice is the resolution.
	Choice ChoiceKind `json:"choice" msgpack:"choice"`
}

// Configuration is a configuration in the repository.
type Configuration struct {
	// Name is the name of the configuration.
	Name string `json:"name" msgpack:"name"`
	// Head is the tip of the configuration.
	Head Commit `json:"head" msgpack:"head"`
	// Current is true for the tracked configuration.
	Current bool `json:"current" msgpack:"current"`
}

// Snapshot is a commit of the live configuration that a range holds.
type Snapshot struct {
	// Range is the key of the range.
	Range string `json:"range" msgpack:"range"`
	// Tag is the name of the tag that marks the snapshot.
	Tag string `json:"tag" msgpack:"tag"`
	// Commit is the snapshot commit.
	Commit Commit `json:"commit" msgpack:"commit"`
	// Configuration is the configuration tracked at the snapshot.
	Configuration string `json:"configuration" msgpack:"configuration"`
	// Uncommitted is the number of uncommitted changes at the snapshot.
	Uncommitted int `json:"uncommitted" msgpack:"uncommitted"`
}

// Author is the person a commit names.
type Author struct {
	// Name is the display name.
	Name string
	// Email is the email address.
	Email string
}
