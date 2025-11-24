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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/device"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
)

var (
	engineerRoleName = "Engineer"
	engineerRole     = role.Role{
		Name:        engineerRoleName,
		Description: "Full access to system configuration, except for user management.",
	}
	engineerPolicies = []policy.Policy{
		{
			Name:   "Engineer Edit Access",
			Effect: policy.EffectAllow,
			Objects: []ontology.ID{
				{Type: label.OntologyType},
				{Type: log.OntologyType},
				{Type: cluster.OntologyType},
				{Type: cluster.NodeOntologyType},
				{Type: channel.OntologyType},
				{Type: group.OntologyType},
				{Type: ranger.OntologyType},
				{Type: framer.OntologyType},
				{Type: ranger.AliasOntologyType},
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
			},
			Actions: []access.Action{access.ActionAll},
		},
		{
			Name:   "Engineer View Access",
			Effect: policy.EffectAllow,
			Objects: []ontology.ID{
				{Type: user.OntologyType},
				{Type: role.OntologyType},
				{Type: policy.OntologyType},
			},
			Actions: []access.Action{access.ActionRetrieve},
		},
	}
)
