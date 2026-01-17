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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
)

// Owner role - Full control of deployment, including user registration and security.
var (
	ownerRoleName = "Owner"
	ownerRole     = role.Role{
		Name:        ownerRoleName,
		Description: "Full control of deployment, including user registration and security.",
	}
	ownerPolicy = policy.Policy{
		Name: ownerRoleName,
	}
)

// Engineer role - Full access to system configuration, except for user management.
var (
	engineerRoleName = "Engineer"
	engineerRole     = role.Role{
		Name:        engineerRoleName,
		Description: "Full access to system configuration, except for user management.",
	}
	engineerPolicies = []policy.Policy{
		{
			Name: "Engineer Edit Access",
		},
		{
			Name: "Engineer View Access",
		},
	}
)

// Operator role - Can view workspaces and control hardware tasks, cannot modify config.
var (
	operatorRoleName = "Operator"
	operatorRole     = role.Role{
		Name:        operatorRoleName,
		Description: "Can view workspaces and visualizations, control hardware and data acquisition tasks. Cannot modify system configuration.",
	}
	operatorPolicies = []policy.Policy{
		{
			Name: "Operator Edit Access",
		},
		{
			Name: "Operator View Access",
		},
	}
)

// Viewer role - View access to all resources.
var (
	viewerRoleName = "Viewer"
	viewerRole     = role.Role{
		Name:        viewerRoleName,
		Description: "View access to all resources.",
	}
	viewerPolicy = policy.Policy{
		Name: viewerRoleName,
	}
)
