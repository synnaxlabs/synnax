// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

type (
	Key  = rack.Key
	Rack = rack.Rack
)

// ServiceConfig is the configuration for the rack API service.
type ServiceConfig struct {
	// Rack stores and retrieves racks.
	//
	// [REQUIRED]
	Rack *rack.Service
	// Task stores and retrieves tasks.
	//
	// [REQUIRED]
	Task *task.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Auth ends the sessions of the racks the service deletes.
	//
	// [REQUIRED]
	Auth *svcauth.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the rack API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Task = override.Nil(c.Task, other.Task)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Auth = override.Nil(c.Auth, other.Auth)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.rack")
	v.NotNil("rack", c.Rack)
	v.NotNil("task", c.Task)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("auth", c.Auth)
	return v.Error()
}

type Service struct {
	access *rbac.Service
	auth   *svcauth.Service
	rack   *rack.Service
	task   *task.Service
	status *status.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		rack:   cfg.Rack,
		task:   cfg.Task,
		status: cfg.Status,
		access: cfg.RBAC,
		auth:   cfg.Auth,
	}, nil
}

type (
	CreateRequest struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
	CreateResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.enforceCreate(ctx, tx, req.Racks); err != nil {
		return CreateResponse{}, err
	}
	if err := s.rack.NewWriter(tx).CreateMany(ctx, &req.Racks); err != nil {
		return CreateResponse{}, err
	}
	if err := s.attachReadableStatuses(ctx, tx, req.Racks); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

func (s *Service) attachReadableStatuses(
	ctx context.Context,
	tx gorp.Tx,
	racks []rack.Rack,
) error {
	enforcer := s.access.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	for i := range racks {
		err := enforcer.Enforce(ctx, access.Request{
			Subject: subject,
			Action:  access.ActionRetrieve,
			Objects: []resource.ID{racks[i].ResourceID()},
		})
		// Create accepts update access, which does not grant read access.
		if errors.Is(err, access.ErrDenied) {
			racks[i].Status = nil
			continue
		}
		if err != nil {
			return err
		}
		if err := s.attachStatuses(ctx, tx, racks[i:i+1]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) attachStatuses(
	ctx context.Context,
	tx gorp.Tx,
	racks []rack.Rack,
) error {
	ids := rack.ResourceIDsFromRacks(racks)
	statuses := make([]rack.Status, 0, len(racks))
	// A status row can be missing, so the resource keeps a nil status.
	if err := s.status.NewRetrieve[struct{}]().
		Where(status.MatchOwners[struct{}](ids...)).
		Entries(&statuses).
		Exec(ctx, tx); err != nil {
		return err
	}
	byOwner := make(map[resource.ID]*rack.Status, len(statuses))
	for i := range statuses {
		byOwner[*statuses[i].Owner] = &statuses[i]
	}
	for i, id := range ids {
		racks[i].Status = byOwner[id]
	}
	return nil
}

// enforceCreate requires the update action on each rack that already exists, and the
// create action when any rack is new, so a subject that may only update racks can
// still save its own.
func (s *Service) enforceCreate(
	ctx context.Context,
	tx gorp.Tx,
	racks []rack.Rack,
) error {
	keys := make([]rack.Key, 0, len(racks))
	for _, r := range racks {
		if !r.Key.IsZero() {
			keys = append(keys, r.Key)
		}
	}
	var existing []rack.Rack
	if len(keys) > 0 {
		if err := s.rack.NewRetrieve().
			Where(rack.MatchKeys(keys...)).
			Entries(&existing).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
	}
	enforcer := s.access.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	if len(existing) > 0 {
		if err := enforcer.Enforce(ctx, access.Request{
			Subject: subject,
			Action:  access.ActionUpdate,
			Objects: rack.ResourceIDsFromRacks(existing),
		}); err != nil {
			return err
		}
	}
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionUpdate,
		Objects: group.CreateObjects(
			racks,
			existing,
			func(r rack.Rack) *group.Key { return r.Group },
		),
	}); err != nil {
		return err
	}
	found := make(set.Set[rack.Key], len(existing))
	for _, r := range existing {
		found.Add(r.Key)
	}
	if !slices.ContainsFunc(
		racks,
		func(r rack.Rack) bool { return !found.Contains(r.Key) },
	) {
		return nil
	}
	return enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeRack}},
	})
}

type (
	RetrieveRequest struct {
		Embedded      *bool      `json:"embedded"       msgpack:"embedded"`
		HostIsNode    *bool      `json:"host_is_node"   msgpack:"host_is_node"`
		SearchTerm    string     `json:"search_term"    msgpack:"search_term"`
		Keys          []rack.Key `json:"keys"           msgpack:"keys"`
		Names         []string   `json:"names"          msgpack:"names"`
		Integration   string     `json:"integration"    msgpack:"integration"`
		Limit         int        `json:"limit"          msgpack:"limit"`
		Offset        int        `json:"offset"         msgpack:"offset"`
		IncludeStatus bool       `json:"include_status" msgpack:"include_status"`
		// Groups keeps only the racks in any of these groups.
		Groups []group.Key `json:"groups" msgpack:"groups"`
		// Grouped keeps only the racks in a group when true, and only the racks in no
		// group when false. Nil applies no filter.
		Grouped *bool `json:"grouped" msgpack:"grouped"`
	}
	RetrieveResponse struct {
		Racks []rack.Rack `json:"racks" msgpack:"racks"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var (
		hasSearch      = len(req.SearchTerm) > 0
		hasKeys        = len(req.Keys) > 0
		hasNames       = len(req.Names) > 0
		hasLimit       = req.Limit > 0
		hasOffset      = req.Offset > 0
		hasIntegration = req.Integration != ""
	)
	resRacks := make([]rack.Rack, 0, len(req.Keys)+len(req.Names))
	q := s.rack.NewRetrieve()
	if hasKeys {
		q = q.Where(rack.MatchKeys(req.Keys...))
	}
	if hasNames {
		q = q.Where(rack.MatchNames(req.Names...))
	}
	if hasSearch {
		q = q.Search(req.SearchTerm)
	}
	if hasLimit {
		q = q.Limit(req.Limit)
	}
	if hasOffset {
		q = q.Offset(req.Offset)
	}
	if req.Embedded != nil {
		q = q.Where(rack.MatchEmbedded(*req.Embedded))
	}
	if req.HostIsNode != nil {
		q = q.Where(rack.MatchNodeIsHost(*req.HostIsNode))
	}
	if hasIntegration {
		q = q.Where(rack.MatchIntegration(req.Integration))
	}
	if len(req.Groups) > 0 {
		q = q.Where(rack.MatchGroups(req.Groups...))
	}
	if req.Grouped != nil {
		q = q.Where(rack.MatchHasGroup(*req.Grouped))
	}
	if err := q.Entries(&resRacks).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}

	if req.IncludeStatus {
		if err := s.attachStatuses(ctx, nil, resRacks); err != nil {
			return RetrieveResponse{}, err
		}
	}

	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: rack.ResourceIDsFromRacks(resRacks),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Racks: resRacks}, nil
}

// RenameRequest names the rack to rename and the name to give it.
type RenameRequest struct {
	// Name is the new name. It must not be empty.
	Name string `json:"name" msgpack:"name"`
	// Key selects the rack to rename.
	Key rack.Key `json:"key" msgpack:"key"`
}

// Rename renames the rack. It requires update access on it.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{req.Key.ResourceID()},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.rack.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type DeleteRequest struct {
	Keys []rack.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the racks and the internal tasks they hold. It refuses a rack that is
// embedded or holds a device or a task that is not internal.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: rack.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	if err := s.task.NewWriter(tx).DeleteRacks(ctx, req.Keys...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).
		DeleteFor(ctx, rack.ResourceIDs(req.Keys)...)
}

// SetGroupRequest moves racks into Group, or out of their group when Group is nil.
type SetGroupRequest struct {
	Group *group.Key `json:"group" msgpack:"group"`
	Keys  []rack.Key `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the racks with the given keys into a group.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	var racks []rack.Rack
	if err := s.rack.NewRetrieve().
		Where(rack.MatchKeys(req.Keys...)).
		Entries(&racks).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.MoveObjects(
			rack.ResourceIDs(req.Keys),
			lo.Map(racks, func(e rack.Rack, _ int) *group.Key { return e.Group }),
			req.Group,
		),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.rack.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}
