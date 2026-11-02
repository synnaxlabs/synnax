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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
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
	// RBAC enforces access control on every request.
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
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	c.Alias = override.Nil(c.Alias, other.Alias)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.channel")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("channel", c.Channel)
	v.NotNil("status", c.Status)
	v.NotNil("alias", c.Alias)
	return v.Error()
}

// Service is the central service for all things Channel related.
type Service struct {
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
	if err := s.internal.NewRetrieve().
		Where(channel.MatchKeys(keys...)).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	return lo.Map(channels, func(ch channel.Channel, _ int) telem.DataType {
		return ch.DataType
	}), nil
}

// RetrieveName implements codec.ChannelResolver, resolving the name of the channel with
// the given key, returning an empty string if no channel with the key exists.
func (s *Service) RetrieveName(ctx context.Context, key channel.Key) string {
	var ch channel.Channel
	if err := s.internal.NewRetrieve().
		Where(channel.MatchKeys(key)).
		Entry(&ch).
		Exec(ctx, nil); err != nil {
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
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeChannel}},
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

// Retrieve retrieves the channels that the request selects. It requires retrieve access
// on every channel and on every alias it fills.
func (s *Service) Retrieve(
	ctx context.Context,
	req AliasedRetrieveRequest,
) (RetrieveResponse, error) {
	resChannels, err := s.retrieve(ctx, req)
	if err != nil {
		return RetrieveResponse{}, err
	}
	res := RetrieveResponse{Channels: translateChannelsForward(resChannels)}
	ids := channel.OntologyIDsFromChannels(resChannels)
	objects := ids
	if req.RangeKey != nil {
		aliased, err := s.fillAliases(ctx, *req.RangeKey, res.Channels)
		if err != nil {
			return RetrieveResponse{}, err
		}
		objects = append(objects, alias.OntologyIDs(*req.RangeKey, aliased)...)
	}
	if req.Request.IncludeStatus {
		statuses, err := s.status.Join[struct{}](ctx, ids)
		if err != nil {
			return RetrieveResponse{}, err
		}
		for i, st := range statuses {
			res.Channels[i].Status = st
		}
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: objects,
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

func (s *Service) retrieve(
	ctx context.Context,
	req AliasedRetrieveRequest,
) ([]channel.Channel, error) {
	r := req.Request
	if req.RangeKey == nil || r.SearchTerm == "" {
		var res []channel.Channel
		q, err := r.Apply(s.internal.NewRetrieve().Entries(&res))
		if err != nil {
			return nil, err
		}
		if err := q.Exec(ctx, nil); err != nil {
			return nil, err
		}
		return res, nil
	}
	// The alias search pages the merged list, so the query carries only the where.
	q, err := RetrieveRequest{Where: r.Where}.Apply(s.internal.NewRetrieve())
	if err != nil {
		return nil, err
	}
	var limit int
	if r.Limit != nil {
		// SearchChannels reads a zero limit as no limit.
		if *r.Limit == 0 {
			return nil, nil
		}
		limit = int(*r.Limit)
	}
	return s.alias.NewReader(nil).SearchChannels(
		ctx, q, *req.RangeKey, r.SearchTerm, limit, int(r.Offset),
	)
}

// fillAliases sets the alias on the range of each channel that has one, and returns the
// keys of those channels.
func (s *Service) fillAliases(
	ctx context.Context,
	rng ranger.Key,
	channels []Channel,
) ([]channel.Key, error) {
	reader := s.alias.NewReader(nil)
	var aliased []channel.Key
	for i, ch := range channels {
		al, err := reader.Retrieve(ctx, rng, ch.Key)
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		channels[i].Alias = &al
		aliased = append(aliased, ch.Key)
	}
	return aliased, nil
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

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	w := s.internal.NewWriter(tx)
	if len(req.Keys) > 0 {
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionDelete,
			Objects: channel.OntologyIDsFromKeys(req.Keys),
		}); err != nil {
			return struct{}{}, err
		}
		if err := w.DeleteMany(ctx, req.Keys, false); err != nil {
			return struct{}{}, err
		}
	}
	if len(req.Names) > 0 {
		res := make([]channel.Channel, 0, len(req.Names))
		if err := s.internal.NewRetrieve().
			Where(channel.MatchNamePatterns(req.Names...)).
			Entries(&res).
			Exec(ctx, tx); err != nil {
			return struct{}{}, err
		}
		if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionDelete,
			Objects: channel.OntologyIDsFromChannels(res),
		}); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, w.DeleteManyByNamePatterns(ctx, req.Names, false)
	}
	return struct{}{}, nil
}

type RenameRequest struct {
	Keys  channel.Keys `json:"keys"  msgpack:"keys"`
	Names []string     `json:"names" msgpack:"names"`
}

func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: channel.OntologyIDsFromKeys(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).RenameMany(
		ctx, req.Keys, req.Names, false,
	)
}

type RetrieveGroupRequest struct{}

type RetrieveGroupResponse struct {
	Group group.Group `json:"group" msgpack:"group"`
}

func (s *Service) RetrieveGroup(
	ctx context.Context,
	_ RetrieveGroupRequest,
) (RetrieveGroupResponse, error) {
	g := s.internal.Group()
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{g.OntologyID()},
	}); err != nil {
		return RetrieveGroupResponse{}, err
	}
	return RetrieveGroupResponse{Group: g}, nil
}
