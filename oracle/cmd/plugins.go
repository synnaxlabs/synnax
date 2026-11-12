// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"github.com/synnaxlabs/oracle/plugin"
	cppactions "github.com/synnaxlabs/oracle/plugin/cpp/actions"
	cppjson "github.com/synnaxlabs/oracle/plugin/cpp/json"
	cpppb "github.com/synnaxlabs/oracle/plugin/cpp/pb"
	cpptypes "github.com/synnaxlabs/oracle/plugin/cpp/types"
	goactions "github.com/synnaxlabs/oracle/plugin/go/actions"
	gofilter "github.com/synnaxlabs/oracle/plugin/go/filter"
	gogroup "github.com/synnaxlabs/oracle/plugin/go/group"
	goimex "github.com/synnaxlabs/oracle/plugin/go/imex"
	gomarshal "github.com/synnaxlabs/oracle/plugin/go/marshal"
	gomigrate "github.com/synnaxlabs/oracle/plugin/go/migrate"
	gopatch "github.com/synnaxlabs/oracle/plugin/go/patch"
	gopb "github.com/synnaxlabs/oracle/plugin/go/pb"
	goquery "github.com/synnaxlabs/oracle/plugin/go/query"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	pbtypes "github.com/synnaxlabs/oracle/plugin/pb/types"
	pyactions "github.com/synnaxlabs/oracle/plugin/py/actions"
	pytypes "github.com/synnaxlabs/oracle/plugin/py/types"
	tsactions "github.com/synnaxlabs/oracle/plugin/ts/actions"
	tsfilter "github.com/synnaxlabs/oracle/plugin/ts/filter"
	tsgroup "github.com/synnaxlabs/oracle/plugin/ts/group"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
)

// buildPluginRegistry returns the canonical set of code-generation plugins shared by
// `oracle sync` and `oracle check`. Sync writes the outputs; check compares them
// against disk. Both must use the same registry, so the set lives here as a single
// source of truth.
//
// The migrate plugin joins the registry for its chain-driven migrate.gen.go emission —
// a pure function of adjacent version files, regenerated on every sync.
func buildPluginRegistry() (*plugin.Registry, error) {
	registry := plugin.NewRegistry()
	for _, p := range []plugin.Plugin{
		gomigrate.New(),
		tstypes.New(tstypes.DefaultOptions()),
		gotypes.New(gotypes.DefaultOptions()),
		pytypes.New(pytypes.DefaultOptions()),
		pyactions.New(pyactions.DefaultOptions()),
		pbtypes.New(pbtypes.DefaultOptions()),
		cpptypes.New(cpptypes.DefaultOptions()),
		cppjson.New(cppjson.DefaultOptions()),
		cpppb.New(cpppb.DefaultOptions()),
		cppactions.New(cppactions.DefaultOptions()),
		gopb.New(gopb.DefaultOptions()),
		goquery.New(goquery.DefaultOptions()),
		gofilter.New(gofilter.DefaultOptions()),
		gogroup.New(gogroup.DefaultOptions()),
		gomarshal.New(gomarshal.DefaultOptions()),
		goactions.New(goactions.DefaultOptions()),
		gopatch.New(gopatch.DefaultOptions()),
		goimex.New(goimex.DefaultOptions()),
		tsactions.New(tsactions.DefaultOptions()),
		tsfilter.New(tsfilter.DefaultOptions()),
		tsgroup.New(tsgroup.DefaultOptions()),
	} {
		if err := registry.Register(p); err != nil {
			return nil, err
		}
	}
	return registry, nil
}
