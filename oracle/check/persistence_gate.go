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

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/pipeline"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// PersistenceGate warns when a persisted type at a version-laid-out path is absent from
// its resource's current version file. Versioning exists to decode stored bytes; a
// persisted type outside the chain would ship bytes no frozen shape records.
type persistenceGate struct {
	// WarningsAsErrors promotes the gate's warnings to errors.
	WarningsAsErrors bool
}

func NewPersistenceGate(warningsAsErrors bool) Checker {
	return persistenceGate{WarningsAsErrors: warningsAsErrors}
}

func (persistenceGate) Name() string { return "persistence" }

func (g persistenceGate) Run(
	ctx context.Context,
	res *pipeline.Result,
	_ Env,
) GateReport {
	start := telem.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	if res.Resolutions == nil {
		r.Elapsed = telem.Since(start)
		return r
	}
	severity := SeverityWarning
	if g.WarningsAsErrors {
		severity = SeverityError
	}
	closure := gotypes.PersistedClosure(res.Resolutions)
	entries, members, err := gotypes.Survey(ctx, res.Resolutions, res.Versions)
	if err != nil {
		r.fail(Finding{Severity: SeverityError, Message: err.Error()})
		r.Elapsed = telem.Since(start)
		return r
	}
	versionedPaths := make(set.Set[string], len(entries))
	for goPath := range entries {
		versionedPaths.Add(goPath)
	}
	for _, t := range res.Resolutions.Types {
		if omit.IsSkipped(t, "go") || output.GetPath(t, "go") == "" {
			continue
		}
		if members.Contains(t.QualifiedName) ||
			!closure.Contains(t.QualifiedName) ||
			!versionedPaths.Contains(output.GetPath(t, "go")) {
			continue
		}
		r.Findings = append(r.Findings, Finding{
			Path:     schemaPathFor(res, t.Namespace),
			Severity: severity,
			Message: t.QualifiedName +
				" is persisted but absent from its resource's current version file",
			FixHint: "declare the type in the current version file so stored " +
				"bytes stay decodable",
		})
		if severity == SeverityError {
			r.Status = StatusFail
		}
	}
	r.Elapsed = telem.Since(start)
	return r
}

// schemaPathFor best-effort maps a namespace to its schema file path.
func schemaPathFor(res *pipeline.Result, namespace string) string {
	for _, rel := range res.Schemas {
		if strings.HasSuffix(rel, "/"+namespace+".oracle") ||
			rel == namespace+".oracle" {
			return rel
		}
	}
	return namespace + ".oracle"
}
