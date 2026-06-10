// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// legacyWorkspaceType is the ontology resource type that became project. Policies
// granting access to specific workspaces stored objects of this type; the rename
// leaves those objects dangling unless they are retyped.
const legacyWorkspaceType = ontology.ResourceType("workspace")

// MigrateWorkspaceObjects retypes every policy object of the legacy "workspace"
// ontology type to "project" after the workspace-to-project rename, so access grants
// on specific workspaces keep applying. A policy's key is independent of its objects,
// so each affected policy is updated in place.
func MigrateWorkspaceObjects(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	iter, err := gorp.WrapReader[Key, Policy](tx).OpenIterator(gorp.IterOptions{})
	if err != nil {
		return err
	}
	var updated []Policy
	for iter.First(); iter.Valid(); iter.Next() {
		p := iter.Value(ctx)
		if p == nil {
			continue
		}
		if next, changed := repointWorkspaceObjects(*p); changed {
			updated = append(updated, next)
		}
	}
	if err = errors.Combine(iter.Error(), iter.Close()); err != nil {
		return err
	}
	w := gorp.WrapWriter[Key, Policy](tx)
	for _, p := range updated {
		if err = w.Set(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// repointWorkspaceObjects returns p with every legacy workspace object retyped to
// project, and whether any object changed. The object slice is copied so the stored
// policy is never mutated in place.
func repointWorkspaceObjects(p Policy) (Policy, bool) {
	objects := make([]ontology.ID, len(p.Objects))
	copy(objects, p.Objects)
	changed := false
	for i := range objects {
		if objects[i].Type == legacyWorkspaceType {
			objects[i].Type = ontology.ResourceTypeProject
			changed = true
		}
	}
	p.Objects = objects
	return p, changed
}
