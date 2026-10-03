// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/x/gorp"
)

// Migration re-encodes stored policies from MessagePack to Orc.
var Migration = gorp.CodecMigration[Key, Policy]("msgpack_to_orc")

// WorkspaceObjectsMigration points the policy objects that name workspaces at projects.
// v0.57 turned every workspace into a project with the same key, but left policies
// naming the old type, so they matched nothing.
var WorkspaceObjectsMigration = gorp.NewEntryMigration(
	"v59_workspace_policy_objects",
	func(_ context.Context, p Policy) (Policy, error) {
		for i, obj := range p.Objects {
			if obj.Type == "workspace" {
				p.Objects[i].Type = ontology.ResourceTypeProject
			}
		}
		return p, nil
	},
)
