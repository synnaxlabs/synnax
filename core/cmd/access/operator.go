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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
)

var (
	operatorRoleName = "Operator"
	operatorRole     = role.Role{
		Name:        operatorRoleName,
		Description: "Can view workspaces and visualizations, control hardware and data acquisition tasks. Cannot modify system configuration.",
	}
	operatorPolicies = []policy.Policy{
		{
			Name:   "Operator Edit Access",
			Effect: policy.EffectAllow,
			Objects: []ontology.ID{
				{Type: framer.OntologyType},
				{Type: ranger.OntologyType},
			},
			Actions: []access.Action{access.ActionAll},
		},
		{
			Name:    "Operator View Access",
			Effect:  policy.EffectAllow,
			Objects: allObjects,
			Actions: []access.Action{access.ActionRetrieve},
		},
	}
)
