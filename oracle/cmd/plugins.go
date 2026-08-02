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
	cppjson "github.com/synnaxlabs/oracle/plugin/cpp/json"
	cpppb "github.com/synnaxlabs/oracle/plugin/cpp/pb"
	cpptypes "github.com/synnaxlabs/oracle/plugin/cpp/types"
	goactions "github.com/synnaxlabs/oracle/plugin/go/actions"
	gomarshal "github.com/synnaxlabs/oracle/plugin/go/marshal"
	gomigrate "github.com/synnaxlabs/oracle/plugin/go/migrate"
	gopb "github.com/synnaxlabs/oracle/plugin/go/pb"
	goquery "github.com/synnaxlabs/oracle/plugin/go/query"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	pbtypes "github.com/synnaxlabs/oracle/plugin/pb/types"
	pytypes "github.com/synnaxlabs/oracle/plugin/py/types"
	tsactions "github.com/synnaxlabs/oracle/plugin/ts/actions"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
)

// buildPluginRegistry returns the canonical set of code-generation plugins
// shared by `oracle sync` and `oracle check`. Sync writes the outputs;
// check compares them against disk. Both must use the same registry, so
// the set lives here as a single source of truth.
//
// The migrate plugin joins the registry for its chain-driven migrate.gen.go
// emission — a pure function of adjacent version files, regenerated on every
// sync. Its bump scaffolding still runs only under `oracle migrate`, which
// alone supplies OldResolutions.
func buildPluginRegistry() *plugin.Registry {
	registry := plugin.NewRegistry()
	_ = registry.Register(gomigrate.New())
	_ = registry.Register(tstypes.New(tstypes.DefaultOptions()))
	_ = registry.Register(gotypes.New(gotypes.DefaultOptions()))
	_ = registry.Register(pytypes.New(pytypes.DefaultOptions()))
	_ = registry.Register(pbtypes.New(pbtypes.DefaultOptions()))
	_ = registry.Register(cpptypes.New(cpptypes.DefaultOptions()))
	_ = registry.Register(cppjson.New(cppjson.DefaultOptions()))
	_ = registry.Register(cpppb.New(cpppb.DefaultOptions()))
	_ = registry.Register(gopb.New(gopb.DefaultOptions()))
	_ = registry.Register(goquery.New(goquery.DefaultOptions()))
	_ = registry.Register(gomarshal.New(gomarshal.DefaultOptions()))
	_ = registry.Register(goactions.New(goactions.DefaultOptions()))
	_ = registry.Register(tsactions.New(tsactions.DefaultOptions()))
	return registry
}
