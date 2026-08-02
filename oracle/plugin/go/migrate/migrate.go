// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate generates the auto-copy migration helpers between adjacent
// schema versions. Each chain version's migrate.gen.go is a pure function of
// the two adjacent version files, regenerated on every sync and verified by
// check. Hand-written transforms live beside it in migrate.go.
package migrate

import (
	"github.com/synnaxlabs/oracle/plugin"
)

// Plugin generates migrate.gen.go files from version chains.
type Plugin struct{}

// New creates the go/migrate plugin.
func New() *Plugin { return &Plugin{} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/migrate" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/types", "go/marshal"} }

// Check implements plugin.Plugin.
func (p *Plugin) Check(*plugin.Request) error { return nil }

// Generate emits the chain-driven migration helpers.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	if req.Versions == nil {
		return resp, nil
	}
	chainOut, err := p.chainFiles(req)
	if err != nil {
		return nil, err
	}
	resp.Files = append(resp.Files, chainOut...)
	return resp, nil
}
