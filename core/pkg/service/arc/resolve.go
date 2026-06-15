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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// resolve populates the Task field of each arc from its associated task in the resource
// graph, and, when includeStatus is set, populates Status from that task's status. Arcs
// that are not deployed to a rack are left with nil Task and Status.
func (s *Service) resolve(
	ctx context.Context,
	tx gorp.Tx,
	arcs []*Arc,
	includeStatus bool,
) error {
	for _, a := range arcs {
		taskKey, ok, err := s.taskKeyFor(ctx, tx, a.Key)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		a.Task = &taskKey
		if !includeStatus {
			continue
		}
		if err = s.resolveStatus(ctx, tx, a, taskKey); err != nil {
			return err
		}
	}
	return nil
}

// taskKeyFor returns the key of the task deployed for the arc with the given key, if any.
func (s *Service) taskKeyFor(
	ctx context.Context,
	tx gorp.Tx,
	key Key,
) (task.Key, bool, error) {
	var children []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(ontology.ResourceTypeTask).
		Entries(&children).
		ExcludeFieldData(true).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return 0, false, err
	}
	if len(children) == 0 {
		return 0, false, nil
	}
	keys, err := task.KeysFromOntologyIDs(ontology.ResourceIDs(children))
	if err != nil {
		return 0, false, err
	}
	return keys[0], true, nil
}

func (s *Service) resolveStatus(
	ctx context.Context,
	tx gorp.Tx,
	a *Arc,
	taskKey task.Key,
) error {
	var ts task.Status
	err := status.NewRetrieve[task.StatusDetails](s.cfg.Status).
		Where(status.MatchKeys[task.StatusDetails](task.OntologyID(taskKey).String())).
		Entry(&ts).
		Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	a.Status = &Status{
		Key:         ts.Key,
		Name:        ts.Name,
		Variant:     ts.Variant,
		Message:     ts.Message,
		Description: ts.Description,
		Time:        ts.Time,
		Details:     StatusDetails{Running: ts.Details.Running},
		Labels:      ts.Labels,
	}
	return nil
}
