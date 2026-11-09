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
	"encoding/json/jsontext"
	"encoding/json/v2"

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
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
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
	// Alias stores and retrieves channel aliases on ranges.
	//
	// [REQUIRED]
	Alias *alias.Service
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
	c.Alias = override.Nil(c.Alias, other.Alias)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.channel")
	v.NotNil("db", c.DB)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("channel", c.Channel)
	v.NotNil("status", c.Status)
	v.NotNil("alias", c.Alias)
	return v.Error()
}

// Service is the central service for all things Channel related.
type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *channel.Service
	status   *status.Service
	alias    *alias.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.DB,
		access:   cfg.RBAC,
		internal: cfg.Channel,
		status:   cfg.Status,
		alias:    cfg.Alias,
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
	translated, err := translateChannelsBackward(req.Channels)
	if err != nil {
		return CreateResponse{}, err
	}
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
		return CreateResponse{}, err
	}
	return CreateResponse{Channels: translateChannelsForward(translated)}, nil
}

// AliasedRetrieveRequest is a RetrieveRequest with an optional range. With a range, the
// Core fills the alias of each channel on it, and a search term also matches those
// aliases, ranked first.
type AliasedRetrieveRequest struct {
	// Request selects the channels.
	Request RetrieveRequest `json:"request" msgpack:"request"`
	// RangeKey is the range whose aliases to fill and search.
	RangeKey *ranger.Key `json:"range_key,omitzero" msgpack:"range_key,omitempty"`
}

// UnmarshalJSONFrom decodes the request and rejects a member that it does not declare,
// so a client that sends a bare channel request fails.
func (r *AliasedRetrieveRequest) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain AliasedRetrieveRequest
	return json.UnmarshalDecode(dec, (*plain)(r), json.RejectUnknownMembers(true))
}

// DecodeMsgpack decodes the request and rejects a member that it does not declare, as
// UnmarshalJSONFrom does.
func (r *AliasedRetrieveRequest) DecodeMsgpack(dec *msgpack.Decoder) error {
	n, err := dec.DecodeMapLen()
	if err != nil {
		return err
	}
	// Each member decodes on its own, because the decode of Request turns off the
	// decoder's unknown-member check when it returns.
	for range n {
		name, err := dec.DecodeString()
		if err != nil {
			return err
		}
		switch name {
		case "request":
			err = dec.Decode(&r.Request)
		case "range_key":
			err = dec.Decode(&r.RangeKey)
		default:
			return errors.Newf("msgpack: unknown field %q", name)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

var _ bind.Named = AliasedRetrieveRequest{}

// Named implements bind.Named. It returns the channels that the top level of Where
// selects by key.
func (r AliasedRetrieveRequest) Named() []resource.ID {
	if r.Request.Where == nil {
		return nil
	}
	return bind.Keys(r.Request.Where.Key, channel.ResourceID)
}

// Retrieve retrieves the channels that the request selects and that the subject can
// retrieve. A channel that the top level of Where names by exact name needs retrieve,
// as a named key does. It fills only the aliases that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req AliasedRetrieveRequest,
) (RetrieveResponse, error) {
	resChannels, err := s.retrieve(ctx, snap, req)
	if err != nil {
		return RetrieveResponse{}, err
	}
	res := RetrieveResponse{Channels: translateChannelsForward(resChannels)}
	if req.RangeKey != nil {
		if err = s.fillAliases(ctx, snap, *req.RangeKey, res.Channels); err != nil {
			return RetrieveResponse{}, err
		}
	}
	if req.Request.IncludeStatus {
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
	return res, nil
}

func (s *Service) retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req AliasedRetrieveRequest,
) ([]channel.Channel, error) {
	if !namesByName(req.Request.Where) {
		return s.match(ctx, snap, req, bind.Filter(snap, channel.Match))
	}
	res, err := s.match(ctx, snap, req, nil)
	if err != nil {
		return nil, err
	}
	if err = snap.Enforce(channel.ResourceIDsFromChannels(res)...); err != nil {
		return nil, err
	}
	return res, nil
}

// match returns the channels that req selects and that filter keeps. A nil filter keeps
// every channel.
func (s *Service) match(
	ctx context.Context,
	snap bind.Snapshot,
	req AliasedRetrieveRequest,
	filter channel.Filter,
) ([]channel.Channel, error) {
	r := req.Request
	if req.RangeKey == nil || r.SearchTerm == "" {
		var res []channel.Channel
		q, err := ApplyRetrieve(r, s.internal.NewRetrieve().Entries(&res))
		if err != nil {
			return nil, err
		}
		if filter != nil {
			q = q.Where(filter)
		}
		if err = q.Exec(ctx, snap); err != nil {
			return nil, err
		}
		return res, nil
	}
	if r.Order != nil {
		return nil, validate.PathedError(
			errors.Wrap(validate.ErrValidation, "a search sorts by relevance"),
			"order",
		)
	}
	if r.After != nil {
		return nil, validate.PathedError(
			errors.Wrap(
				validate.ErrValidation,
				"a search cannot resume after a position",
			),
			"after",
		)
	}
	// The alias search limits the merged list, so the query carries only the where.
	q, err := ApplyRetrieve(RetrieveRequest{Where: r.Where}, s.internal.NewRetrieve())
	if err != nil {
		return nil, err
	}
	if filter != nil {
		q = q.Where(filter)
	}
	var limit int
	if r.Limit != nil {
		// SearchChannels reads a zero limit as no limit.
		if *r.Limit == 0 {
			return nil, nil
		}
		limit = int(*r.Limit)
	}
	rng := *req.RangeKey
	return s.alias.NewReader(snap).SearchChannels(
		ctx,
		q,
		rng,
		r.SearchTerm,
		func(ch channel.Key) bool { return snap.Allowed(alias.ResourceID(rng, ch)) },
		limit,
	)
}

// namesByName reports whether the top level of where names channels by exact name.
// Clients name channels by name as they do by key, so each match needs retrieve.
func namesByName(where *channel.Where) bool {
	return where != nil && where.Name != nil &&
		(where.Name.Eq != nil || len(where.Name.In) > 0)
}

// fillAliases sets the alias on the range of each channel that has one the subject can
// retrieve.
func (s *Service) fillAliases(
	ctx context.Context,
	snap bind.Snapshot,
	rng ranger.Key,
	channels []Channel,
) error {
	reader := s.alias.NewReader(snap)
	for i, ch := range channels {
		if !snap.Allowed(alias.ResourceID(rng, ch.Key)) {
			continue
		}
		al, err := reader.Retrieve(ctx, rng, ch.Key)
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		channels[i].Alias = &al
	}
	return nil
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
func translateChannelsBackward(
	channels []Channel,
) ([]channel.Channel, error) {
	translated := make([]channel.Channel, len(channels))
	for i, ch := range channels {
		tCh := channel.Channel{
			Name:        ch.Name,
			Leaseholder: node.Key(ch.Leaseholder),
			DataType:    ch.DataType,
			IsIndex:     ch.IsIndex,
			LocalIndex:  ch.Index.LocalKey(),
			LocalKey:    ch.Key.LocalKey(),
			Virtual:     ch.Virtual,
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
	return translated, nil
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
	computed, err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
	if err != nil {
		return DispatchResponse{}, err
	}
	return DispatchResponse{Actions: computed}, nil
}
