// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil holds helpers that specs of several services share.
package testutil

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"runtime"

	"github.com/synnaxlabs/x/errors"
)

// Case is a filter case: the records, by alias, that Where matches.
type Case[W any] struct {
	// Name names the case in spec output.
	Name string `json:"name"`
	// Where is the filter under test.
	Where W `json:"where"`
	// Expected holds the aliases of the records that Where matches, in any order.
	Expected []string `json:"expected"`
	// ServerComputed is for the TypeScript matches. The Core computes every case.
	ServerComputed bool `json:"server_computed"`
}

// Vectors holds the records of one entity, keyed by alias, and its filter cases.
type Vectors[R, W any] struct {
	// Records holds the records to create before each case, keyed by alias.
	Records map[string]R `json:"records"`
	// Cases are the filters to run against Records.
	Cases []Case[W] `json:"cases"`
}

// ReadVectors reads the filter vectors of the entity, such as "ranges", which the
// TypeScript client shares. It returns an error if the entity has no vectors or a
// vector has an unknown member.
func ReadVectors[R, W any](entity string) (Vectors[R, W], error) {
	var v Vectors[R, W]
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "../../../testdata/filter/vectors.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return v, err
	}
	var entities map[string]jsontext.Value
	if err := json.Unmarshal(b, &entities); err != nil {
		return v, err
	}
	raw, ok := entities[entity]
	if !ok {
		return v, errors.Newf("no filter vectors for %s", entity)
	}
	if err := json.Unmarshal(raw, &v, json.RejectUnknownMembers(true)); err != nil {
		return Vectors[R, W]{}, err
	}
	return v, nil
}
