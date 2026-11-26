// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Key = channel.Key

// ServiceConfig is the configuration for the channel API service.
type ServiceConfig struct {
	// DB opens the snapshots that codec lookups read.
	//
	// [REQUIRED]
	DB *gorp.DB
	// RBAC checks that the subject can read the inputs of each calculated channel that
	// a create writes.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Channel stores and retrieves channels.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.channel")
	v.NotNil("db", c.DB)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("channel", c.Channel)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service is the central service for all things Channel related.
type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *channel.Service
	status   *status.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.DB,
		access:   cfg.RBAC,
		internal: cfg.Channel,
		status:   cfg.Status,
	}, nil
}

var _ codec.ChannelResolver = (*Service)(nil)

// RetrieveDataTypes implements codec.ChannelResolver, resolving the data types of the
// channels with the given keys so that a dynamic framer codec can resolve channel
// metadata through the API layer. Resolution intentionally bypasses access control, as
// it runs on the per-frame encode/decode hot path.
func (s *Service) RetrieveDataTypes(
	ctx context.Context,
	keys channel.Keys,
) ([]telem.DataType, error) {
	var channels []channel.Channel
	if err := s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.internal.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Entries(&channels).
			Exec(ctx, snap)
	}); err != nil {
		return nil, err
	}
	dataTypes := make(map[channel.Key]telem.DataType, len(channels))
	for _, ch := range channels {
		dataTypes[ch.Key()] = ch.DataType
	}
	return lo.Map(keys, func(k channel.Key, _ int) telem.DataType {
		return dataTypes[k]
	}), nil
}

// RetrieveName implements codec.ChannelResolver, resolving the name of the channel with
// the given key, returning an empty string if no channel with the key exists.
func (s *Service) RetrieveName(ctx context.Context, key channel.Key) string {
	var ch channel.Channel
	if err := s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return s.internal.NewRetrieve().
			Where(channel.MatchKeys(key)).
			Entry(&ch).
			Exec(ctx, snap)
	}); err != nil {
		return ""
	}
	return ch.Name
}

// CreateRequest is a request to create a Channel in the cluster.
type CreateRequest struct {
	// Channels is a template for the Channels to create.
	Channels             []Channel `json:"channels"                msgpack:"channels"`
	RetrieveIfNameExists bool      `json:"retrieve_if_name_exists" msgpack:"retrieve_if_name_exists"`
}

// CreateResponse is the response returned after a set of channels have
// successfully been created in the cluster.
type CreateResponse struct {
	Channels []Channel `json:"channels" msgpack:"channels"`
}

var _ bind.Returns = CreateResponse{}

// Returned implements bind.Returns, so a create that returns an existing channel needs
// retrieve on it.
func (r CreateResponse) Returned() []resource.ID {
	return lo.Map(r.Channels, func(ch Channel, _ int) resource.ID {
		return channel.ResourceID(ch.Key)
	})
}

// Create creates a Channel based on the parameters given in the request.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	translated := translateChannelsBackward(req.Channels)
	for i := range translated {
		translated[i].Internal = false
	}
	inputs, err := s.internal.CalculationInputs(ctx, tx, translated)
	if err != nil {
		return CreateResponse{}, err
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: framer.ResourceIDs(inputs),
	}); err != nil {
		return CreateResponse{}, err
	}
	opts := []channel.CreateOption{}
	if req.RetrieveIfNameExists {
		opts = append(opts, channel.RetrieveIfNameExists())
	}
	if err := s.internal.NewWriter(tx).
		CreateMany(ctx, &translated, opts...); err != nil {
		return CreateResponse{}, s.describe(ctx, tx, err)
	}
	return CreateResponse{Channels: translateChannelsForward(translated)}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the channels that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, channel.ResourceID)
}

// Retrieve retrieves the channels that the request selects and that the subject can
// retrieve. A channel that the top level of Where names by exact name or path needs
// retrieve, as a named key does.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	resChannels, err := s.retrieve(ctx, snap, req)
	if err != nil {
		return RetrieveResponse{}, err
	}
	res := RetrieveResponse{Channels: translateChannelsForward(resChannels)}
	if req.IncludeStatus {
		statuses, err := s.status.Join[struct{}](
			ctx,
			snap,
			channel.ResourceIDsFromChannels(resChannels),
		)
		if err != nil {
			return RetrieveResponse{}, err
		}
		for i, st := range statuses {
			res.Channels[i].Status = st
		}
	}
	if req.IncludePath {
		if err = fillPaths(ctx, snap, resChannels, res.Channels); err != nil {
			return RetrieveResponse{}, err
		}
	}
	return res, nil
}

// fillPaths sets the path of each of channels, the API forms of stored, when the
// subject can retrieve every folder on it.
func fillPaths(
	ctx context.Context,
	snap bind.Snapshot,
	stored []channel.Channel,
	channels []Channel,
) error {
	paths, err := channel.Paths(ctx, snap, stored)
	if err != nil {
		return err
	}
	// Each folder is checked once, though many paths pass it.
	checked, allowed := make(set.Set[group.Key]), make(set.Set[group.Key])
	visible := func(g group.Group) bool {
		if !checked.Contains(g.Key) {
			checked.Add(g.Key)
			if snap.Allowed(g.ResourceID()) {
				allowed.Add(g.Key)
			}
		}
		return allowed.Contains(g.Key)
	}
	text := channel.PathTexts(paths)
	for i, p := range paths {
		if lo.EveryBy(p.Folders, visible) {
			channels[i].Path = &text[i]
		}
	}
	return nil
}

func (s *Service) retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) ([]channel.Channel, error) {
	var res []channel.Channel
	if req.Where != nil {
		where := channel.NormalizeWhere(*req.Where)
		req.Where = &where
	}
	if req.Strict {
		if err := s.refuseAmbiguous(ctx, snap, req.Where); err != nil {
			return nil, err
		}
	}
	q, err := ApplyRetrieve(
		req,
		s.internal.NewRetrieve().AllowFolders(snap.Allowed).Entries(&res),
	)
	if err != nil {
		return nil, err
	}
	named := namedExactly(req.Where)
	if !named {
		q = q.Where(bind.Filter(snap, channel.Match))
	}
	if err = q.Exec(ctx, snap); err != nil {
		return nil, err
	}
	if named {
		if err = snap.Enforce(channel.ResourceIDsFromChannels(res)...); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// refuseAmbiguous returns a multiple results error when a name of the strict where is
// held by several channels, every holder counted, with the paths that the subject can
// see. It returns a validation error when where holds a match other than a key, name,
// or path, at the top level or as the branches of one or.
func (s *Service) refuseAmbiguous(
	ctx context.Context,
	snap bind.Snapshot,
	where *channel.Where,
) error {
	names, err := strictNames(where)
	if err != nil {
		return err
	}
	_, err = s.internal.KeysByNames(ctx, snap, names)
	return s.internal.DescribeAmbiguity(ctx, snap, err, func(
		context.Context,
	) (func(resource.ID) bool, error) {
		return snap.Allowed, nil
	})
}

// strictNames returns the names that a strict where matches, and a validation error
// when where holds a match other than a key, name, or path.
func strictNames(where *channel.Where) ([]string, error) {
	refuse := validate.PathedError(
		validate.Errorf("a strict retrieve takes only key, name, and path matches"),
		"where",
	)
	if where == nil {
		return nil, refuse
	}
	branches := []channel.Where{*where}
	if len(where.Or) > 0 {
		if !lookupsOnly(*where) || where.Key != nil || where.Name != nil ||
			where.Path != nil {
			return nil, refuse
		}
		branches = where.Or
	}
	var names []string
	for _, b := range branches {
		if !lookupsOnly(b) || len(b.Or) > 0 || !looksUp(b) {
			return nil, refuse
		}
		if n := b.Name; n != nil {
			if n.Contains != nil || n.StartsWith != nil || n.Regex != nil ||
				n.CaseInsensitive {
				return nil, refuse
			}
			names = append(names, n.In...)
			if n.Eq != nil {
				names = append(names, *n.Eq)
			}
		}
	}
	return names, nil
}

// looksUp reports whether w holds a key, name, or path to look up.
func looksUp(w channel.Where) bool {
	return w.Key != nil && (w.Key.Eq != nil || len(w.Key.In) > 0) ||
		w.Name != nil && (w.Name.Eq != nil || len(w.Name.In) > 0) ||
		w.Path != nil && (w.Path.Eq != nil || len(w.Path.In) > 0)
}

// lookupsOnly reports whether w holds no match but key, name, path, and or.
func lookupsOnly(w channel.Where) bool {
	return len(w.And) == 0 && w.Not == nil && w.Leaseholder == nil &&
		w.DataType == nil && w.IsIndex == nil && w.Internal == nil &&
		w.Virtual == nil && len(w.Groups) == 0 && w.Grouped == nil
}

// namedExactly reports whether the top level of where names channels by exact name or
// path. Clients name channels by name or path as they do by key, so each match needs
// retrieve.
func namedExactly(where *channel.Where) bool {
	if where == nil {
		return false
	}
	return where.Name != nil && (where.Name.Eq != nil || len(where.Name.In) > 0) ||
		where.Path != nil && (where.Path.Eq != nil || len(where.Path.In) > 0)
}

func translateChannelsForward(channels []channel.Channel) []Channel {
	translated := make([]Channel, len(channels))
	for i, ch := range channels {
		translated[i] = Channel{
			Key:         ch.Key(),
			Name:        ch.Name,
			Leaseholder: ch.Leaseholder,
			DataType:    ch.DataType,
			IsIndex:     ch.IsIndex,
			Index:       ch.Index(),
			Virtual:     ch.Virtual,
			Concurrency: ch.Concurrency,
			Internal:    ch.Internal,
			Expression:  ch.Expression,
			Operations:  ch.Operations,
			Group:       ch.Group,
		}
	}
	return translated
}

// translateChannelsBackward translates a slice of api channel structs to a
// slice of internal channel structs.
func translateChannelsBackward(channels []Channel) []channel.Channel {
	translated := make([]channel.Channel, len(channels))
	for i, ch := range channels {
		tCh := channel.Channel{
			Name:        ch.Name,
			Leaseholder: ch.Leaseholder,
			DataType:    ch.DataType,
			IsIndex:     ch.IsIndex,
			LocalIndex:  ch.Index.LocalKey(),
			LocalKey:    ch.Key.LocalKey(),
			Virtual:     ch.Virtual,
			Concurrency: ch.Concurrency,
			Internal:    ch.Internal,
			Expression:  ch.Expression,
			Operations:  ch.Operations,
			Group:       ch.Group,
		}
		if ch.IsIndex {
			tCh.LocalIndex = tCh.LocalKey
		}

		translated[i] = tCh
	}
	return translated
}

// CountLimitedRequest is a request for the count of the channels that count against the
// license channel limit.
type CountLimitedRequest struct{}

var _ bind.Named = CountLimitedRequest{}

// Named implements bind.Named. The count covers every channel, so it names the channel
// type.
func (CountLimitedRequest) Named() []resource.ID {
	return []resource.ID{{Type: resource.TypeChannel}}
}

// CountLimited returns how many channels count against the license channel limit (not
// internal, not virtual).
func (s *Service) CountLimited(
	context.Context,
	bind.Snapshot,
	CountLimitedRequest,
) (CountLimitedResponse, error) {
	return CountLimitedResponse{Count: s.internal.CountLimited()}, nil
}

type DeleteRequest struct {
	Keys  channel.Keys `json:"keys"  msgpack:"keys"`
	Names []string     `json:"names" msgpack:"names"`
}

// Delete deletes the channels that the request names by key or by name. Deleting an
// internal channel is a validation error. A name that several channels hold returns a
// query.MultipleResults error.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	named, err := s.internal.KeysByNames(ctx, tx, req.Names)
	if err != nil {
		return struct{}{}, s.describe(ctx, tx, err)
	}
	return struct{}{}, s.internal.NewWriter(tx).
		DeleteMany(ctx, slices.Concat(req.Keys, named), false)
}

// describe returns err as the subject of ctx sees it, through
// channel.Service.DescribeAmbiguity.
func (s *Service) describe(ctx context.Context, tx gorp.Tx, err error) error {
	return s.internal.DescribeAmbiguity(ctx, tx, err, func(
		ctx context.Context,
	) (func(resource.ID) bool, error) {
		return s.access.NewEnforcer(tx).
			Allowed(ctx, auth.GetSubject(ctx), access.ActionRetrieve)
	})
}

// DispatchRequest carries an action sequence to apply to a single channel. DispatchKey
// is a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[channel.Key, channel.Action]

// DispatchResponse carries the actions that the Core added to a dispatch, such as a set
// of the data type that an expression edit infers.
type DispatchResponse = actions.DispatchResponse[channel.Action]

// Dispatch applies the action sequence to the target channel atomically. Subscribers to
// the channel action channel receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (DispatchResponse, error) {
	computed, err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
	if err != nil {
		return DispatchResponse{}, s.describe(ctx, tx, err)
	}
	return DispatchResponse{Actions: computed}, nil
}
