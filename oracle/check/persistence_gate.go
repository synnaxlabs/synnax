// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check

import (
	"context"
	"strings"
	"time"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin/domain"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// PersistenceGate warns when a type declares @go version but is not reachable
// from any @go marshal root. Versioning exists to decode stored bytes; a
// never-persisted type pays the version ceremony for nothing and should be
// left unversioned (struct-level @go version on persisted siblings).
type PersistenceGate struct {
	// WarningsAsErrors promotes the gate's warnings to errors.
	WarningsAsErrors bool
}

// NewPersistenceGate constructs a persistence gate.
func NewPersistenceGate(warningsAsErrors bool) *PersistenceGate {
	return &PersistenceGate{WarningsAsErrors: warningsAsErrors}
}

func (PersistenceGate) Name() string { return "persistence" }

func (g PersistenceGate) Run(_ context.Context, p *pipeline.Result, _ Env) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	if p.Resolutions == nil {
		r.Elapsed = time.Since(start)
		return r
	}
	severity := SeverityWarning
	if g.WarningsAsErrors {
		severity = SeverityError
	}
	closure := gotypes.PersistedClosure(p.Resolutions)
	versionedPaths := make(set.Set[string])
	for _, t := range p.Resolutions.Types {
		if domain.HasExprFromType(t, "go", "version") {
			versionedPaths.Add(output.GetPath(t, "go"))
		}
	}
	// A type referenced by a versioned sibling at its own path — through any
	// field, omitted or not — must ride the version layout regardless of
	// persistence: its Go declaration cannot leave the package without an
	// import cycle. Such types are exempt from the never-persisted warning.
	siblingRef := make(set.Set[string])
	for _, t := range p.Resolutions.Types {
		if !domain.HasExprFromType(t, "go", "version") {
			continue
		}
		path := output.GetPath(t, "go")
		gotypes.WalkTypeRefs(t, p.Resolutions, func(target resolution.Type) {
			if output.GetPath(target, "go") == path {
				siblingRef.Add(target.QualifiedName)
			}
		})
	}
	for _, t := range p.Resolutions.Types {
		if omit.IsSkipped(t, "go") || output.GetPath(t, "go") == "" {
			continue
		}
		versioned := domain.HasExprFromType(t, "go", "version")
		persisted := closure.Contains(t.QualifiedName)
		pinned := gotypes.VersionPinned(t)
		var f Finding
		switch {
		case versioned && pinned && persisted:
			f = Finding{
				Path:     schemaPathFor(p, t.Namespace),
				Severity: severity,
				Message:  t.QualifiedName + " pins its @go version but is persisted",
				FixHint:  "remove the pinned marker; persistence alone requires the version",
			}
		case versioned && !persisted && (pinned || siblingRef.Contains(t.QualifiedName)):
			continue
		case versioned && !persisted:
			f = Finding{
				Path:     schemaPathFor(p, t.Namespace),
				Severity: severity,
				Message:  t.QualifiedName + " declares @go version but is never persisted",
				FixHint:  "declare @go version struct-level on persisted types only",
			}
		case !versioned && persisted && versionedPaths.Contains(output.GetPath(t, "go")):
			f = Finding{
				Path:     schemaPathFor(p, t.Namespace),
				Severity: severity,
				Message: t.QualifiedName +
					" is persisted but lacks @go version at a versioned path",
				FixHint: "add @go version to the type so stored bytes stay decodable",
			}
		default:
			continue
		}
		r.Findings = append(r.Findings, f)
		if severity == SeverityError {
			r.Status = StatusFail
		}
	}
	r.Elapsed = time.Since(start)
	return r
}

// schemaPathFor best-effort maps a namespace to its schema file path.
func schemaPathFor(p *pipeline.Result, namespace string) string {
	for _, rel := range p.Schemas {
		if strings.HasSuffix(rel, "/"+namespace+".oracle") ||
			rel == namespace+".oracle" {
			return rel
		}
	}
	return namespace + ".oracle"
}
