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
	gomarshal "github.com/synnaxlabs/oracle/plugin/go/marshal"
	gopb "github.com/synnaxlabs/oracle/plugin/go/pb"
	goquery "github.com/synnaxlabs/oracle/plugin/go/query"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	pbtypes "github.com/synnaxlabs/oracle/plugin/pb/types"
	pytypes "github.com/synnaxlabs/oracle/plugin/py/types"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
)

// buildPluginRegistry returns the canonical set of code-generation plugins
// shared by `oracle sync` and `oracle check`. Sync writes the outputs;
// check compares them against disk. Both must use the same registry, so
// the set lives here as a single source of truth.
//
// The migrate plugin is intentionally excluded; it has its own command
// (`oracle migrate`) with bespoke snapshot handling and is not part of the
// regular generation pipeline.
func buildPluginRegistry() *plugin.Registry {
	registry := plugin.NewRegistry()
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
	return registry
}
