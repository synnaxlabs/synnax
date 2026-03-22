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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
)

// allObjects is the complete list of ontology types used for permission definitions.
var allObjects = []ontology.ID{
	{Type: ontology.TypeLabel},
	{Type: ontology.TypeLog},
	{Type: ontology.TypeCluster},
	{Type: ontology.TypeNode},
	{Type: ontology.TypeChannel},
	{Type: ontology.TypeGroup},
	{Type: ontology.TypeRange},
	{Type: ontology.TypeFramer},
	{Type: ontology.TypeRangeAlias},
	{Type: ontology.TypeUser},
	{Type: ontology.TypeWorkspace},
	{Type: ontology.TypeSchematic},
	{Type: ontology.TypeLineplot},
	{Type: ontology.TypeRack},
	{Type: ontology.TypeDevice},
	{Type: ontology.TypeTask},
	{Type: ontology.TypeTable},
	{Type: ontology.TypeArc},
	{Type: ontology.TypeSchematicSymbol},
	{Type: ontology.TypeStatus},
	{Type: ontology.TypeRole},
	{Type: ontology.TypePolicy},
	{Type: ontology.TypeBuiltIn},
	{Type: ontology.TypeView},
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
				{Type: ontology.TypeLabel},
				{Type: ontology.TypeLog},
				{Type: ontology.TypeCluster},
				{Type: ontology.TypeNode},
				{Type: ontology.TypeChannel},
				{Type: ontology.TypeGroup},
				{Type: ontology.TypeRange},
				{Type: ontology.TypeFramer},
				{Type: ontology.TypeRangeAlias},
				{Type: ontology.TypeWorkspace},
				{Type: ontology.TypeSchematic},
				{Type: ontology.TypeLineplot},
				{Type: ontology.TypeRack},
				{Type: ontology.TypeDevice},
				{Type: ontology.TypeTask},
				{Type: ontology.TypeTable},
				{Type: ontology.TypeArc},
				{Type: ontology.TypeSchematicSymbol},
				{Type: ontology.TypeStatus},
				{Type: ontology.TypeView},
			},
			Actions:  access.AllActions,
			Internal: true,
		},
		{
			Name: "Engineer View Access",
			Objects: []ontology.ID{
				{Type: ontology.TypeUser},
				{Type: ontology.TypeRole},
				{Type: ontology.TypePolicy},
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
				{Type: ontology.TypeFramer},
				{Type: ontology.TypeRange},
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
