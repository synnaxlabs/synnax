// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/telem"
)

// resolveFieldValue resolves a field path to its actual value.
func resolveFieldValue(ctx context.Context, params EnforceParams, target string, field []string) (any, bool) {
	switch target {
	case "request":
		return resolveRequestField(params.Request, field)
	case "resource":
		return resolveResourceField(ctx, params, field)
	case "subject":
		return resolveSubjectField(ctx, params, field)
	case "system":
		return resolveSystemField(field)
	default:
		return nil, false
	}
}

func resolveRequestField(req access.Request, field []string) (any, bool) {
	if len(field) == 0 || req.Context == nil {
		return nil, false
	}
	value, ok := req.Context[field[0]]
	return value, ok
}

func resolveResourceField(ctx context.Context, params EnforceParams, field []string) (any, bool) {
	if len(field) == 0 || len(params.Request.Objects) == 0 {
		return nil, false
	}
	var resources []ontology.Resource
	if err := params.Ontology.NewRetrieve().
		WhereIDs(params.Request.Objects[0]).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil || len(resources) == 0 {
		return nil, false
	}
	if resources[0].Data == nil {
		return nil, false
	}
	return getNestedField(resources[0].Data, field)
}

func resolveSubjectField(ctx context.Context, params EnforceParams, field []string) (any, bool) {
	if len(field) == 0 {
		return params.Request.Subject, true
	}
	var resources []ontology.Resource
	if err := params.Ontology.NewRetrieve().
		WhereIDs(params.Request.Subject).
		Entries(&resources).
		Exec(ctx, params.Tx); err != nil || len(resources) == 0 {
		return nil, false
	}
	if resources[0].Data == nil {
		return nil, false
	}
	return getNestedField(resources[0].Data, field)
}

func resolveSystemField(field []string) (any, bool) {
	if len(field) == 0 {
		return nil, false
	}
	switch field[0] {
	case "current_time":
		return telem.Now(), true
	default:
		return nil, false
	}
}

func getNestedField(data any, path []string) (any, bool) {
	if len(path) == 0 {
		return data, true
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil, false
	}
	value, ok := dataMap[path[0]]
	if !ok {
		return nil, false
	}
	if len(path) == 1 {
		return value, true
	}
	return getNestedField(value, path[1:])
}
