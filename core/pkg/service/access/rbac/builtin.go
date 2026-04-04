// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
)

var allObjects = []ontology.ID{
	{Type: ontology.ResourceTypeLabel},
	{Type: ontology.ResourceTypeLog},
	{Type: ontology.ResourceTypeCluster},
	{Type: ontology.ResourceTypeNode},
	{Type: ontology.ResourceTypeChannel},
	{Type: ontology.ResourceTypeGroup},
	{Type: ontology.ResourceTypeRange},
	{Type: ontology.ResourceTypeFramer},
	{Type: ontology.ResourceTypeRangeAlias},
	{Type: ontology.ResourceTypeUser},
	{Type: ontology.ResourceTypeWorkspace},
	{Type: ontology.ResourceTypeSchematic},
	{Type: ontology.ResourceTypeLineplot},
	{Type: ontology.ResourceTypeRack},
	{Type: ontology.ResourceTypeDevice},
	{Type: ontology.ResourceTypeTask},
	{Type: ontology.ResourceTypeTable},
	{Type: ontology.ResourceTypeArc},
	{Type: ontology.ResourceTypeSchematicSymbol},
	{Type: ontology.ResourceTypeStatus},
	{Type: ontology.ResourceTypeRole},
	{Type: ontology.ResourceTypePolicy},
	{Type: ontology.ResourceTypeBuiltin},
	{Type: ontology.ResourceTypeView},
}

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
				{Type: ontology.ResourceTypeLabel},
				{Type: ontology.ResourceTypeLog},
				{Type: ontology.ResourceTypeCluster},
				{Type: ontology.ResourceTypeNode},
				{Type: ontology.ResourceTypeChannel},
				{Type: ontology.ResourceTypeGroup},
				{Type: ontology.ResourceTypeRange},
				{Type: ontology.ResourceTypeFramer},
				{Type: ontology.ResourceTypeRangeAlias},
				{Type: ontology.ResourceTypeWorkspace},
				{Type: ontology.ResourceTypeSchematic},
				{Type: ontology.ResourceTypeLineplot},
				{Type: ontology.ResourceTypeRack},
				{Type: ontology.ResourceTypeDevice},
				{Type: ontology.ResourceTypeTask},
				{Type: ontology.ResourceTypeTable},
				{Type: ontology.ResourceTypeArc},
				{Type: ontology.ResourceTypeSchematicSymbol},
				{Type: ontology.ResourceTypeStatus},
				{Type: ontology.ResourceTypeView},
			},
			Actions:  access.AllActions,
			Internal: true,
		},
		{
			Name: "Engineer View Access",
			Objects: []ontology.ID{
				{Type: ontology.ResourceTypeUser},
				{Type: ontology.ResourceTypeRole},
				{Type: ontology.ResourceTypePolicy},
			},
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
	}
)

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
				{Type: ontology.ResourceTypeFramer},
				{Type: ontology.ResourceTypeRange},
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
