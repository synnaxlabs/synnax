// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package builtin

import (
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// allObjects returns every object type the built-in roles cover: the static resource
// types plus the task config record types the caller derives from the config
// registry.
func allObjects(taskConfigObjects []ontology.ID) []ontology.ID {
	return append([]ontology.ID{
		{Type: ontology.ResourceTypeLabel},
		{Type: ontology.ResourceTypeLog},
		{Type: ontology.ResourceTypeNode},
		{Type: ontology.ResourceTypeChannel},
		{Type: ontology.ResourceTypeGroup},
		{Type: ontology.ResourceTypeRange},
		{Type: ontology.ResourceTypeFramer},
		{Type: ontology.ResourceTypeRangeAlias},
		{Type: ontology.ResourceTypeUser},
		{Type: ontology.ResourceTypeProject},
		{Type: ontology.ResourceTypePanel},
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
	}, taskConfigObjects...)
}

var (
	ownerRoleName = "Owner"
	ownerRole     = role.Role{
		Name:        ownerRoleName,
		Description: "Full control of deployment, including user registration and security.",
		Internal:    true,
	}
)

func ownerPolicy(taskConfigObjects []ontology.ID) policy.Policy {
	return policy.Policy{
		Name:     ownerRoleName,
		Objects:  allObjects(taskConfigObjects),
		Actions:  access.AllActions,
		Internal: true,
	}
}

var (
	engineerRoleName = "Engineer"
	engineerRole     = role.Role{
		Name:        engineerRoleName,
		Description: "Full access to system configuration, except for user management.",
		Internal:    true,
	}
)

func engineerPolicies(taskConfigObjects []ontology.ID) []policy.Policy {
	return []policy.Policy{
		{
			Name: "Engineer Edit Access",
			Objects: append([]ontology.ID{
				{Type: ontology.ResourceTypeLabel},
				{Type: ontology.ResourceTypeLog},
				{Type: ontology.ResourceTypeNode},
				{Type: ontology.ResourceTypeChannel},
				{Type: ontology.ResourceTypeGroup},
				{Type: ontology.ResourceTypeRange},
				{Type: ontology.ResourceTypeFramer},
				{Type: ontology.ResourceTypeRangeAlias},
				{Type: ontology.ResourceTypeProject},
				{Type: ontology.ResourceTypePanel},
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
			}, taskConfigObjects...),
			Actions:  access.AllActions,
			Internal: true,
		},
		{
			Name: "Engineer View Access",
			Objects: []ontology.ID{
				{Type: ontology.ResourceTypeUser},
				{Type: ontology.ResourceTypeRole},
				{Type: ontology.ResourceTypePolicy},
				{Type: ontology.ResourceTypeBuiltin},
			},
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
	}
}

var (
	hostRoleName = "Host"
	hostRole     = role.Role{
		Name:        hostRoleName,
		Description: "For machines running the Synnax driver. Full access to hardware and task configuration.",
		Internal:    true,
	}
)

func hostPolicies(taskConfigObjects []ontology.ID) []policy.Policy {
	return []policy.Policy{
		{
			Name: "Host Edit Access",
			Objects: append([]ontology.ID{
				{Type: ontology.ResourceTypeRange},
				{Type: ontology.ResourceTypeRack},
				{Type: ontology.ResourceTypeDevice},
				{Type: ontology.ResourceTypeTask},
				{Type: ontology.ResourceTypeArc},
				{Type: ontology.ResourceTypeStatus},
			}, taskConfigObjects...),
			Actions:  access.AllActions,
			Internal: true,
		},
		{
			Name:     "Host Channel View Access",
			Objects:  []ontology.ID{{Type: ontology.ResourceTypeChannel}},
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
		{
			Name:    "Host Framer Access",
			Objects: []ontology.ID{{Type: ontology.ResourceTypeFramer}},
			Actions: []access.Action{
				access.ActionCreate,
				access.ActionRetrieve,
			},
			Internal: true,
		},
	}
}

var (
	operatorRoleName = "Operator"
	operatorRole     = role.Role{
		Name:        operatorRoleName,
		Description: "Can view projects and visualizations, control hardware and data acquisition tasks. Cannot modify system configuration.",
		Internal:    true,
	}
)

func operatorPolicies(taskConfigObjects []ontology.ID) []policy.Policy {
	return []policy.Policy{
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
			Objects:  allObjects(taskConfigObjects),
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
	}
}

var (
	viewerRoleName = "Viewer"
	viewerRole     = role.Role{
		Name:        viewerRoleName,
		Description: "View access to all resources.",
		Internal:    true,
	}
)

func viewerPolicy(taskConfigObjects []ontology.ID) policy.Policy {
	return policy.Policy{
		Name:     viewerRoleName,
		Objects:  allObjects(taskConfigObjects),
		Actions:  []access.Action{access.ActionRetrieve},
		Internal: true,
	}
}
