// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Resources returns the resource.Provider of projects.
func (s *Service) Resources() *resource.TableProvider[Key, Project] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Project]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(p *Project) string { return p.Name },
	})
}

// ResourceID returns the resource ID of the project with key k.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeProject, Key: k.String()}
}

// ResourceIDs returns the resource ID of the project with each key.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromProjects returns the resource ID of each project.
func ResourceIDsFromProjects(projects []Project) []resource.ID {
	return lo.Map(projects, func(p Project, _ int) resource.ID {
		return p.ResourceID()
	})
}

// KeyFromResourceID converts id to the Key of the project it identifies. It returns an
// error wrapping validate.ErrValidation if id is not a project or its key is not a
// valid UUID. Callers that resolve a caller-supplied field scope the returned error to
// that field with validate.PathedError.
func KeyFromResourceID(id resource.ID) (Key, error) {
	if id.Type != resource.TypeProject {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "must be a project, got %q", id.Type,
		)
	}
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "invalid project key %q: %v", id.Key, err,
		)
	}
	return key, nil
}
