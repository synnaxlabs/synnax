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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var allObjects = []resource.ID{
	{Type: resource.TypeLabel},
	{Type: resource.TypeLog},
	{Type: resource.TypeNode},
	{Type: resource.TypeChannel},
	{Type: resource.TypeGroup},
	{Type: resource.TypeRange},
	{Type: resource.TypeFramer},
	{Type: resource.TypeRangeAlias},
	{Type: resource.TypeUser},
	{Type: resource.TypeProject},
	{Type: resource.TypePanel},
	{Type: resource.TypeSchematic},
	{Type: resource.TypeLineplot},
	{Type: resource.TypeRack},
	{Type: resource.TypeDevice},
	{Type: resource.TypeEthercatMaster},
	{Type: resource.TypeEthercatSlave},
	{Type: resource.TypeHTTPServer},
	{Type: resource.TypeLabjackDevice},
	{Type: resource.TypeModbusServer},
	{Type: resource.TypeNiChassis},
	{Type: resource.TypeNiDevice},
	{Type: resource.TypeOpcuaServer},
	{Type: resource.TypeTask},
	{Type: resource.TypeTable},
	{Type: resource.TypeArc},
	{Type: resource.TypeArcDirectory},
	{Type: resource.TypeSchematicSymbol},
	{Type: resource.TypeStatus},
	{Type: resource.TypeRole},
	{Type: resource.TypePolicy},
	{Type: resource.TypeBuiltin},
	{Type: resource.TypeView},
	{Type: resource.TypeSecret},
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
			Objects: []resource.ID{
				{Type: resource.TypeLabel},
				{Type: resource.TypeLog},
				{Type: resource.TypeNode},
				{Type: resource.TypeChannel},
				{Type: resource.TypeGroup},
				{Type: resource.TypeRange},
				{Type: resource.TypeFramer},
				{Type: resource.TypeRangeAlias},
				{Type: resource.TypeProject},
				{Type: resource.TypePanel},
				{Type: resource.TypeSchematic},
				{Type: resource.TypeLineplot},
				{Type: resource.TypeRack},
				{Type: resource.TypeDevice},
				{Type: resource.TypeEthercatMaster},
				{Type: resource.TypeEthercatSlave},
				{Type: resource.TypeHTTPServer},
				{Type: resource.TypeLabjackDevice},
				{Type: resource.TypeModbusServer},
				{Type: resource.TypeNiChassis},
				{Type: resource.TypeNiDevice},
				{Type: resource.TypeOpcuaServer},
				{Type: resource.TypeTask},
				{Type: resource.TypeTable},
				{Type: resource.TypeArc},
				{Type: resource.TypeArcDirectory},
				{Type: resource.TypeSchematicSymbol},
				{Type: resource.TypeStatus},
				{Type: resource.TypeView},
				{Type: resource.TypeSecret},
			},
			Actions:  access.AllActions,
			Internal: true,
		},
		{
			Name: "Engineer View Access",
			Objects: []resource.ID{
				{Type: resource.TypeUser},
				{Type: resource.TypeRole},
				{Type: resource.TypePolicy},
				{Type: resource.TypeBuiltin},
			},
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
	}
)

var (
	driverRoleName = "Driver"
	driverRole     = role.Role{
		Name:        driverRoleName,
		Description: "For racks running the Synnax Driver. Full access to hardware and task configuration.",
		Internal:    true,
	}
	driverPolicies = []policy.Policy{
		{
			Name: "Driver Edit Access",
			Objects: []resource.ID{
				{Type: resource.TypeRange},
				{Type: resource.TypeDevice},
				{Type: resource.TypeEthercatMaster},
				{Type: resource.TypeEthercatSlave},
				{Type: resource.TypeHTTPServer},
				{Type: resource.TypeLabjackDevice},
				{Type: resource.TypeModbusServer},
				{Type: resource.TypeNiChassis},
				{Type: resource.TypeNiDevice},
				{Type: resource.TypeOpcuaServer},
				{Type: resource.TypeTask},
				{Type: resource.TypeArc},
				{Type: resource.TypeStatus},
			},
			Actions:  access.AllActions,
			Internal: true,
		},
		{
			Name:     "Driver Rack Access",
			Objects:  []resource.ID{{Type: resource.TypeRack}},
			Actions:  []access.Action{access.ActionRetrieve, access.ActionUpdate},
			Internal: true,
		},
		{
			Name:     "Driver Channel View Access",
			Objects:  []resource.ID{{Type: resource.TypeChannel}},
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
		{
			Name:     "Driver Secret View Access",
			Objects:  []resource.ID{{Type: resource.TypeSecret}},
			Actions:  []access.Action{access.ActionRetrieve},
			Internal: true,
		},
		{
			Name:    "Driver Framer Access",
			Objects: []resource.ID{{Type: resource.TypeFramer}},
			Actions: []access.Action{
				access.ActionCreate,
				access.ActionRetrieve,
			},
			Internal: true,
		},
	}
)

// formerNames maps a built-in role or policy name to the name it was stored under
// before a rename, so provisioning renames the stored record instead of adding one.
var formerNames = map[string]string{
	driverRoleName:               "Host",
	"Driver Edit Access":         "Host Edit Access",
	"Driver Channel View Access": "Host Channel View Access",
	"Driver Framer Access":       "Host Framer Access",
}

var (
	operatorRoleName = "Operator"
	operatorRole     = role.Role{
		Name:        operatorRoleName,
		Description: "Can view projects and visualizations, control hardware and data acquisition tasks. Cannot modify system configuration.",
		Internal:    true,
	}
	operatorPolicies = []policy.Policy{
		{
			Name: "Operator Edit Access",
			Objects: []resource.ID{
				{Type: resource.TypeFramer},
				{Type: resource.TypeRange},
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
