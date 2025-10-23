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
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

var (
	rootRoleName = "Root"
	rootPolicy   = policy.Policy{
		Name:   rootRoleName,
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
		},
		Actions:  []access.Action{access.ActionAll},
		Internal: true,
	}
)

func ProvisionRootRole(
	ctx context.Context,
	tx gorp.Tx,
	service *rbac.Service,
) (uuid.UUID, error) {
	var pol policy.Policy
	if err := service.Policy.NewRetrieve().
		WhereInternal(true).
		WhereNames(rootPolicy.Name).
		Exec(ctx, tx); errors.Skip(err, query.NotFound) != nil {
		return uuid.Nil, err
	}
	if pol.Key == uuid.Nil {
		pol = rootPolicy
		if err := service.Policy.NewWriter(tx).Create(ctx, &pol); err != nil {
			return uuid.Nil, err
		}
	}
	var rol role.Role
	if err := service.Role.NewRetrieve().
		WhereInternal(true).
		WhereName(rootRoleName).Exec(ctx, tx); errors.Skip(err, query.NotFound) != nil {
		return uuid.Nil, err
	}
	if rol.Key == uuid.Nil {
		rol = role.Role{
			Name:        rootRoleName,
			Description: "Root Permissions",
			Policies:    []uuid.UUID{pol.Key},
		}
		if err := service.Role.NewWriter(tx).Create(ctx, &rol); err != nil {
			return uuid.Nil, err
		}
	}
	return rol.Key, nil
}
