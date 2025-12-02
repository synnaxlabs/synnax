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

var allObjects = []ontology.ID{
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
	{Type: ontology.BuiltInType},
}

// ProvisionResult contains the keys of all provisioned built-in roles.
type ProvisionResult struct {
	OwnerKey    uuid.UUID
	EngineerKey uuid.UUID
	OperatorKey uuid.UUID
	ViewerKey   uuid.UUID
}

func Provision(
	ctx context.Context,
	tx gorp.Tx,
	service *rbac.Service,
) (ProvisionResult, error) {
	var result ProvisionResult
	var err error

	if result.ViewerKey, err = provisionRole(ctx, viewerRole, []policy.Policy{viewerPolicy}, tx, service); err != nil {
		return ProvisionResult{}, err
	}
	if result.OperatorKey, err = provisionRole(ctx, operatorRole, operatorPolicies, tx, service); err != nil {
		return ProvisionResult{}, err
	}
	if result.EngineerKey, err = provisionRole(ctx, engineerRole, engineerPolicies, tx, service); err != nil {
		return ProvisionResult{}, err
	}
	if result.OwnerKey, err = provisionRole(ctx, ownerRole, []policy.Policy{ownerPolicy}, tx, service); err != nil {
		return ProvisionResult{}, err
	}
	return result, nil
}

func provisionRole(
	ctx context.Context,
	rol role.Role,
	policies []policy.Policy,
	tx gorp.Tx,
	service *rbac.Service,
) (uuid.UUID, error) {
	policyKeys := make([]uuid.UUID, 0, len(policies))

	// Create or retrieve all policies
	for i := range policies {
		pol := &policies[i]
		if err := service.Policy.NewRetrieve().
			WhereNames(pol.Name).
			Entry(pol).
			Exec(ctx, tx); errors.Skip(err, query.NotFound) != nil {
			return uuid.Nil, err
		}
		if pol.Key == uuid.Nil {
			if err := service.Policy.NewWriter(tx, true).Create(ctx, pol); err != nil {
				return uuid.Nil, err
			}
		}
		policyKeys = append(policyKeys, pol.Key)
	}

	// Create or retrieve the role
	if err := service.Role.NewRetrieve().
		WhereName(rol.Name).
		Entry(&rol).
		Exec(ctx, tx); errors.Skip(err, query.NotFound) != nil {
		return uuid.Nil, err
	}
	if rol.Key == uuid.Nil {
		w := service.Role.NewWriter(tx, true)
		if err := w.Create(ctx, &rol); err != nil {
			return uuid.Nil, err
		}
		// Associate all policies with the role
		if err := service.Policy.NewWriter(tx, true).SetOnRole(ctx, rol.Key, policyKeys...); err != nil {
			return uuid.Nil, err
		}
	}
	return rol.Key, nil
}
