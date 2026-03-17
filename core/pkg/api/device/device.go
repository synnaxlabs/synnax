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
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	device   *device.Service
	status   *status.Service
	ontology *ontology.Ontology
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.Distribution.DB,
		device:   cfg.Service.Device,
		status:   cfg.Service.Status,
		access:   cfg.Service.RBAC,
		ontology: cfg.Distribution.Ontology,
	}, nil
}

// Device wraps the service-layer Device with an optional Parent ontology ID
// used during creation to establish parent-child relationships.
type Device struct {
	device.Device
	Parent *ontology.ID `json:"parent" msgpack:"parent"`
}

type CreateRequest struct {
	Devices []Device `json:"devices" msgpack:"devices"`
}

type CreateResponse struct {
	Devices []Device `json:"devices" msgpack:"devices"`
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (res CreateResponse, _ error) {
	svcDevices := make([]device.Device, len(req.Devices))
	for i, d := range req.Devices {
		svcDevices[i] = d.Device
	}
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: device.OntologyIDsFromDevices(svcDevices),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.device.NewWriter(tx)
		for i, d := range req.Devices {
			var parent ontology.ID
			if d.Parent != nil {
				parent = *d.Parent
			}
			if err := w.Create(ctx, d.Device, parent); err != nil {
				return err
			}
			req.Devices[i].Device = d.Device
			// Populate the response parent with the resolved parent ID.
			// If no explicit parent was provided, the device defaults to its rack.
			resolvedParent := d.Rack.OntologyID()
			if d.Parent != nil && !d.Parent.IsZero() {
				resolvedParent = *d.Parent
			}
			req.Devices[i].Parent = &resolvedParent
		}
		res.Devices = req.Devices
		return nil
	})
}

type RetrieveRequest struct {
	SearchTerm     string     `json:"search_term" msgpack:"search_term"`
	Keys           []string   `json:"keys" msgpack:"keys"`
	Names          []string   `json:"names" msgpack:"names"`
	Makes          []string   `json:"makes" msgpack:"makes"`
	Models         []string   `json:"models" msgpack:"models"`
	Locations      []string   `json:"locations" msgpack:"locations"`
	Racks          []rack.Key `json:"racks" msgpack:"racks"`
	Limit          int        `json:"limit" msgpack:"limit"`
	Offset         int        `json:"offset" msgpack:"offset"`
	IgnoreNotFound bool       `json:"ignore_not_found" msgpack:"ignore_not_found"`
	IncludeStatus  bool       `json:"include_status" msgpack:"include_status"`
	IncludeParent  bool       `json:"include_parent" msgpack:"include_parent"`
}

type RetrieveResponse struct {
	Devices []Device `json:"devices" msgpack:"devices"`
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (res RetrieveResponse, _ error) {
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
		q = q.WhereKeys(req.Keys...)
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if hasMakes {
		q = q.WhereMakes(req.Makes...)
	}
	if hasLocations {
		q = q.WhereLocations(req.Locations...)
	}
	if hasModels {
		q = q.WhereModels(req.Models...)
	}
	if hasRacks {
		q = q.WhereRacks(req.Racks...)
	}
	var svcDevices []device.Device
	retErr := q.Entries(&svcDevices).Exec(ctx, nil)

	if req.IncludeStatus {
		statuses := make([]device.Status, 0, len(svcDevices))
		if err := status.NewRetrieve[device.StatusDetails](s.status).
			WhereKeys(ontology.IDsToKeys(device.OntologyIDsFromDevices(svcDevices))...).
			Entries(&statuses).
			Exec(ctx, nil); err != nil {
			return res, err
		}
		for i, stat := range statuses {
			svcDevices[i].Status = &stat
		}
	}

	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: device.OntologyIDsFromDevices(svcDevices),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	if retErr != nil && req.IgnoreNotFound {
		retErr = errors.Skip(retErr, query.ErrNotFound)
	}

	res.Devices = make([]Device, len(svcDevices))
	for i, d := range svcDevices {
		res.Devices[i].Device = d
		if req.IncludeParent {
			var parents []ontology.Resource
			if err := s.ontology.NewRetrieve().
				WhereIDs(device.OntologyID(d.Key)).
				TraverseTo(ontology.ParentsTraverser).
				Limit(1).
				ExcludeFieldData(true).
				Entries(&parents).
				Exec(ctx, nil); err != nil {
				return RetrieveResponse{}, err
			}
			if len(parents) > 0 {
				id := parents[0].ID
				res.Devices[i].Parent = &id
			}
		}
	}
	return res, retErr
}

type DeleteRequest struct {
	Keys []string `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (res types.Nil, _ error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: device.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.device.NewWriter(tx)
		for _, k := range req.Keys {
			if err := w.Delete(ctx, k); err != nil {
				return err
			}
		}
		return nil
	})
}
