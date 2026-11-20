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
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
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
	// Calculation stores the calculation of each channel that the API gives an
	// expression.
	//
	// [REQUIRED]
	Calculation *calculation.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the channel API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	c.Calculation = override.Nil(c.Calculation, other.Calculation)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.channel")
	v.NotNil("db", c.DB)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("channel", c.Channel)
	v.NotNil("status", c.Status)
	v.NotNil("calculation", c.Calculation)
	return v.Error()
}

// Service is the central service for all things Channel related.
type Service struct {
	db           *gorp.DB
	access       *rbac.Service
	internal     *channel.Service
	status       *status.Service
	calculations *calculation.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:           cfg.DB,
		access:       cfg.RBAC,
		internal:     cfg.Channel,
		status:       cfg.Status,
		calculations: cfg.Calculation,
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
	inputs, err := s.calculations.Inputs(ctx, tx, translated)
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
	if err := s.calculations.NewWriter(tx).
		CreateChannels(ctx, &translated, req.RetrieveIfNameExists); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{
		Channels: translateChannelsForward(translated, s.internal.IsBuiltIn),
	}, nil
}

// statusOwners returns the owner of the status of each of channels: the calculation
// that writes it, or the channel itself.
func (s *Service) statusOwners(
	ctx context.Context,
	snap gorp.Snapshot,
	channels []channel.Channel,
) ([]resource.ID, error) {
	calcs, err := s.calculations.Outputs(
		ctx,
		snap,
		channel.KeysFromChannels(channels)...,
	)
	if err != nil {
		return nil, err
	}
	owners := make([]resource.ID, len(channels))
	for i, ch := range channels {
		owners[i] = channel.ResourceID(ch.Key())
		if c, ok := calcs[ch.Key()]; ok {
			owners[i] = calculation.ResourceID(c.Key)
		}
	}
	return owners, nil
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
// retrieve. A channel that the top level of Where names by exact name needs retrieve,
// as a named key does.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	resChannels, err := s.retrieve(ctx, snap, req)
	if err != nil {
		return RetrieveResponse{}, err
	}
	res := RetrieveResponse{
		Channels: translateChannelsForward(resChannels, s.internal.IsBuiltIn),
	}
	if req.IncludeStatus {
		owners, err := s.statusOwners(ctx, snap, resChannels)
		if err != nil {
			return RetrieveResponse{}, err
		}
		statuses, err := s.status.Join[struct{}](ctx, snap, owners)
		if err != nil {
			return RetrieveResponse{}, err
		}
		for i, st := range statuses {
			res.Channels[i].Status = st
		}
	}
	if req.IncludeUses {
		if err = s.fillUses(ctx, snap, res.Channels); err != nil {
			return RetrieveResponse{}, err
		}
	}
	return res, nil
}

// fillUses sets the uses of each channel, leaving out the records the subject cannot
// retrieve. A channel with no readable uses gets empty uses, so the response tells it
// apart from a channel whose uses were not asked for.
func (s *Service) fillUses(
	ctx context.Context,
	snap bind.Snapshot,
	channels []Channel,
) error {
	keys := lo.Map(channels, func(ch Channel, _ int) channel.Key { return ch.Key })
	uses, err := s.internal.Uses(ctx, snap, keys)
	if err != nil {
		return err
	}
	type holder struct {
		id      resource.ID
		allowed bool
	}
	holders := make(map[gorp.Target]holder)
	total := 0
	for _, u := range uses {
		total += len(u)
	}
	// One array backs the uses of every channel.
	backing := make([]Use, 0, total)
	for i := range channels {
		start := len(backing)
		for _, u := range uses[i] {
			h, ok := holders[u.Holder]
			if !ok {
				h.id = taskconfig.RecordOf(u.Holder)
				h.allowed = snap.Allowed(h.id)
				holders[u.Holder] = h
			}
			if h.allowed {
				backing = append(backing, Use{ID: h.id, Direction: u.Direction})
			}
		}
		channels[i].Uses = backing[start:len(backing):len(backing)]
	}
	return nil
}

func (s *Service) retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) ([]channel.Channel, error) {
	var res []channel.Channel
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res))
	if err != nil {
		return nil, err
	}
	named := namesByName(req.Where)
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

// namesByName reports whether the top level of where names channels by exact name.
// Clients name channels by name as they do by key, so each match needs retrieve.
func namesByName(where *channel.Where) bool {
	return where != nil && where.Name != nil &&
		(where.Name.Eq != nil || len(where.Name.In) > 0)
}

func translateChannelsForward(
	channels []channel.Channel,
	builtIn func(channel.Key) bool,
) []Channel {
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
			Internal:    builtIn(ch.Key()),
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

// RetrieveBuiltInsRequest is a request for the keys of the built-in channels. It names
// no record, so any subject may send it.
type RetrieveBuiltInsRequest struct{}

// RetrieveBuiltIns returns the keys of the channels that Core services declare for
// their work.
func (s *Service) RetrieveBuiltIns(
	context.Context,
	bind.Snapshot,
	RetrieveBuiltInsRequest,
) (RetrieveBuiltInsResponse, error) {
	return RetrieveBuiltInsResponse{Keys: s.internal.BuiltIns()}, nil
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
// internal channel is a validation error.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	w := s.internal.NewWriter(tx)
	if len(req.Keys) > 0 {
		if err := w.DeleteMany(ctx, req.Keys, false); err != nil {
			return struct{}{}, err
		}
	}
	if len(req.Names) > 0 {
		return struct{}{}, w.DeleteManyByNames(ctx, req.Names, false)
	}
	return struct{}{}, nil
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
	var prev, next channel.Channel
	if err := s.internal.NewRetrieve().
		Where(channel.MatchKeys(req.Key)).
		Entry(&prev).
		Exec(ctx, tx); err != nil {
		return DispatchResponse{}, err
	}
	computed, err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
	if err != nil {
		return DispatchResponse{}, err
	}
	if err = s.internal.NewRetrieve().
		Where(channel.MatchKeys(req.Key)).
		Entry(&next).
		Exec(ctx, tx); err != nil {
		return DispatchResponse{}, err
	}
	// Only an edit of the expression or the operations writes the calculation, so the
	// other edits keep what the calculation API wrote.
	if next.Expression != prev.Expression ||
		!slices.Equal(next.Operations, prev.Operations) {
		if err = s.calculations.NewWriter(tx).SetFromChannel(ctx, next); err != nil {
			return DispatchResponse{}, err
		}
	}
	return DispatchResponse{Actions: computed}, nil
}
