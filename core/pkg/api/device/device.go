// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the device API service.
type ServiceConfig struct {
	// Device stores and retrieves devices.
	//
	// [REQUIRED]
	Device *device.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Ontology stores the relationships between resources.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the device API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Device = override.Nil(c.Device, other.Device)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.device")
	v.NotNil("device", c.Device)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("ontology", c.Ontology)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	device   *device.Service
	status   *status.Service
	ontology *ontology.Ontology
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		device:   cfg.Device,
		status:   cfg.Status,
		access:   cfg.RBAC,
		ontology: cfg.Ontology,
	}, nil
}

type (
	CreateRequest struct {
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
	CreateResponse struct {
		Devices []device.Device `json:"devices" msgpack:"devices"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeDevice}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.device.NewWriter(tx).CreateMany(ctx, &req.Devices); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

type RetrieveRequest struct {
	SearchTerm     string       `json:"search_term"      msgpack:"search_term"`
	Keys           []device.Key `json:"keys"             msgpack:"keys"`
	Names          []string     `json:"names"            msgpack:"names"`
	Makes          []string     `json:"makes"            msgpack:"makes"`
	Models         []string     `json:"models"           msgpack:"models"`
	Locations      []string     `json:"locations"        msgpack:"locations"`
	Racks          []rack.Key   `json:"racks"            msgpack:"racks"`
	Limit          int          `json:"limit"            msgpack:"limit"`
	Offset         int          `json:"offset"           msgpack:"offset"`
	IgnoreNotFound bool         `json:"ignore_not_found" msgpack:"ignore_not_found"`
	IncludeStatus  bool         `json:"include_status"   msgpack:"include_status"`
	IncludeParent  bool         `json:"include_parent"   msgpack:"include_parent"`
}

type RetrieveResponse struct {
	Devices []device.Device `json:"devices" msgpack:"devices"`
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var (
		hasSearch    = len(req.SearchTerm) > 0
		hasKeys      = len(req.Keys) > 0
		hasNames     = len(req.Names) > 0
		hasMakes     = len(req.Makes) > 0
		hasLimit     = req.Limit > 0
		hasOffset    = req.Offset > 0
		hasLocations = len(req.Locations) > 0
		hasModels    = len(req.Models) > 0
		hasRacks     = len(req.Racks) > 0
	)
	q := s.device.NewRetrieve()
	if hasKeys {
		q = q.Where(device.MatchKeys(req.Keys...))
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if hasNames {
		q = q.Where(device.MatchNames(req.Names...))
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if hasMakes {
		q = q.Where(device.MatchMakes(req.Makes...))
	}
	if hasLocations {
		q = q.Where(device.MatchLocations(req.Locations...))
	}
	if hasModels {
		q = q.Where(device.MatchModels(req.Models...))
	}
	if hasRacks {
		q = q.Where(device.MatchRacks(req.Racks...))
	}
	var res RetrieveResponse
	retErr := q.Entries(&res.Devices).Exec(ctx, nil)

	if req.IncludeStatus {
		ids := device.OntologyIDsFromDevices(res.Devices)
		statuses := make([]device.Status, 0, len(res.Devices))
		// A status row can be missing, so the resource keeps a nil status.
		if err := errors.Skip(s.status.NewRetrieve[device.StatusDetails]().
			Where(status.MatchKeys[device.StatusDetails](ontology.IDsToKeys(ids)...)).
			Entries(&statuses).
			Exec(ctx, nil), query.ErrNotFound); err != nil {
			return RetrieveResponse{}, err
		}
		statusByKey := make(map[status.Key]*device.Status, len(statuses))
		for i := range statuses {
			statusByKey[statuses[i].Key] = &statuses[i]
		}
		for i, id := range ids {
			res.Devices[i].Status = statusByKey[id.String()]
		}
	}

	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: device.OntologyIDsFromDevices(res.Devices),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	if retErr != nil && req.IgnoreNotFound {
		retErr = errors.Skip(retErr, query.ErrNotFound)
	}
	if req.IncludeParent {
		for i, d := range res.Devices {
			var parent ontology.Resource
			err := s.ontology.NewRetrieve().
				WhereIDs(d.OntologyID()).
				TraverseTo(ontology.ParentsTraverser).
				Limit(1).
				ExcludeFieldData(true).
				Entry(&parent).
				Exec(ctx, nil)
			if err != nil {
				if !errors.Is(err, query.ErrNotFound) {
					return RetrieveResponse{}, err
				}
				continue
			}
			pid := ontology.ID(parent.ID)
			res.Devices[i].Parent = &pid
		}
	}
	if retErr != nil {
		return RetrieveResponse{}, retErr
	}
	return res, nil
}

// RenameRequest names the device to rename and the name to give it.
type RenameRequest struct {
	// Name is the new name. It must not be empty.
	Name string `json:"name" msgpack:"name"`
	// Key selects the device to rename.
	Key device.Key `json:"key" msgpack:"key"`
}

// Rename renames the device. It requires update access on it.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{device.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.device.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type DeleteRequest struct {
	Keys []device.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: device.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	w := s.device.NewWriter(tx)
	for _, k := range req.Keys {
		if err := w.Delete(ctx, k); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}
