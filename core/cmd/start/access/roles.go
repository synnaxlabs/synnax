// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
)

// allObjects is the complete list of ontology types used for permission definitions.
var allObjects = []ontology.ID{
	{Type: label.OntologyType},
	{Type: log.OntologyType},
	{Type: cluster.OntologyType},
	{Type: cluster.OntologyTypeNode},
	{Type: channel.OntologyType},
	{Type: group.OntologyType},
	{Type: ranger.OntologyType},
	{Type: framer.OntologyType},
	{Type: alias.OntologyType},
	{Type: user.OntologyType},
	{Type: workspace.OntologyType},
	{Type: schematic.OntologyType},
	{Type: lineplot.OntologyType},
	{Type: rack.OntologyType},
	{Type: device.OntologyType},
	{Type: task.OntologyType},
	{Type: table.OntologyType},
	{Type: arc.OntologyType},
	{Type: symbol.OntologyType},
	{Type: status.OntologyType},
	{Type: role.OntologyType},
	{Type: policy.OntologyType},
	{Type: ontology.TypeBuiltIn},
	{Type: view.OntologyType},
}

// Owner role - Full control of deployment, including user registration and security.
var (
	ownerRoleName = "Owner"
	ownerRole     = role.Role{
		Name:        ownerRoleName,
		Description: "Full control of deployment, including user registration and security.",
		Internal:    true,
	}
	ownerPolicy = policy.Policy{
		Name:     ownerRoleName,
		Objects:  allObjects,
		Actions:  access.AllActions,
		Internal: true,
	}
)

// Engineer role - Full access to system configuration, except for user management.
var (
	engineerRoleName = "Engineer"
	engineerRole     = role.Role{
		Name:        engineerRoleName,
		Description: "Full access to system configuration, except for user management.",
		Internal:    true,
	}
	engineerPolicies = []policy.Policy{
		{
			Name: "Engineer Edit Access",
			Objects: []ontology.ID{
				{Type: label.OntologyType},
				{Type: log.OntologyType},
				{Type: cluster.OntologyType},
				{Type: cluster.OntologyTypeNode},
				{Type: channel.OntologyType},
				{Type: group.OntologyType},
				{Type: ranger.OntologyType},
				{Type: framer.OntologyType},
				{Type: alias.OntologyType},
				{Type: workspace.OntologyType},
				{Type: schematic.OntologyType},
				{Type: lineplot.OntologyType},
				{Type: rack.OntologyType},
				{Type: device.OntologyType},
				{Type: task.OntologyType},
				{Type: table.OntologyType},
				{Type: arc.OntologyType},
				{Type: symbol.OntologyType},
				{Type: status.OntologyType},
				{Type: view.OntologyType},
			},
			Actions:  access.AllActions,
			Internal: true,
		},
		{
			Name: "Engineer View Access",
			Objects: []ontology.ID{
				{Type: user.OntologyType},
				{Type: role.OntologyType},
				{Type: policy.OntologyType},
			},
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
	}
)

// Operator role - Can view workspaces and control hardware tasks, cannot modify config.
var (
	operatorRoleName = "Operator"
	operatorRole     = role.Role{
		Name:        operatorRoleName,
		Description: "Can view workspaces and visualizations, control hardware and data acquisition tasks. Cannot modify system configuration.",
		Internal:    true,
	}
	operatorPolicies = []policy.Policy{
		{
			Name: "Operator Edit Access",
			Objects: []ontology.ID{
				{Type: framer.OntologyType},
				{Type: ranger.OntologyType},
			},
			Actions:  access.AllActions,
			Internal: true,
		},
		{
			Name:     "Operator View Access",
			Objects:  allObjects,
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
	}
)

// Viewer role - View access to all resources.
var (
	viewerRoleName = "Viewer"
	viewerRole     = role.Role{
		Name:        viewerRoleName,
		Description: "View access to all resources.",
		Internal:    true,
	}
	viewerPolicy = policy.Policy{
		Name:     viewerRoleName,
		Objects:  allObjects,
		Actions:  []access.Action{access.ActionRetrieve},
		Internal: true,
	}
)
