// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
)

var (
	ownerRoleName = "Owner"
	ownerRole     = role.Role{
		Name:        ownerRoleName,
		Description: "Full control of deployment, including user registration and security",
	}
	ownerPolicy = policy.Policy{
		Name:    ownerRoleName,
		Effect:  policy.EffectAllow,
		Objects: allObjects,
		Actions: []access.Action{access.ActionAll},
	}
)
