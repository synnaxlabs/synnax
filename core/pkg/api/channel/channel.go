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
	"strconv"

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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
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
	// Calculation stores the calculation of each channel that the API gives an
	// expression.
	//
	// [REQUIRED]
	Calculation *calculation.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

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
	cfg, err := config.New(ServiceConfig{}, cfgs...)
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
	var (
		outputs   channel.Keys
		unsaved   []calculation.Unsaved
		unsavedAt []int
	)
	for i, ch := range req.Channels {
		if ch.Expression != "" {
			ops, err := calculation.FromChannelOperations(ch.Operations)
			if err != nil {
				return CreateResponse{}, validate.PathedError(
					err,
					strconv.Itoa(i),
					"operations",
				)
			}
			unsaved = append(unsaved, calculation.Unsaved{
				Output:     translated[i],
				Expression: ch.Expression,
				Operations: ops,
			})
			unsavedAt = append(unsavedAt, i)
		} else if ch.Key.LocalKey() != 0 {
			outputs = append(outputs, translated[i].Key())
		}
	}
	inputs, err := s.calculations.Inputs(ctx, tx, outputs, unsaved)
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
	stored, err := s.createChannels(
		ctx,
		tx,
		translated,
		unsaved,
		unsavedAt,
		req.RetrieveIfNameExists,
	)
	if err != nil {
		return CreateResponse{}, s.describe(ctx, tx, err)
	}
	calcs, err := s.calculations.Outputs(ctx, tx, virtualKeys(stored)...)
	if err != nil {
		return CreateResponse{}, err
	}
	enforcer := s.access.NewEnforcer(tx)
	shown, err := s.shown(calcs, func(ids []resource.ID) (bool, error) {
		err := enforcer.Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionRetrieve,
			Objects: ids,
		})
		if errors.Is(err, access.ErrDenied) {
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Channels: s.translateForward(stored, calcs, shown)}, nil
}

// statusOwners returns the owner of the status of each of channels: the calculation
// that writes it, or the channel itself.
func statusOwners(
	channels []channel.Channel,
	calcs map[channel.Key]calculation.Calculation,
) []resource.ID {
	owners := make([]resource.ID, len(channels))
	for i, ch := range channels {
		owners[i] = channel.ResourceID(ch.Key())
		if c, ok := calcs[ch.Key()]; ok {
			owners[i] = calculation.ResourceID(c.Key)
		}
	}
	return owners
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
// retrieve,
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
	// The embedded snapshot converts to an interface with no allocation.
	calcs, err := s.calculations.Outputs(
		ctx,
		snap.Snapshot,
		virtualKeys(resChannels)...,
	)
	if err != nil {
		return RetrieveResponse{}, err
	}
	var shown map[channel.Key]calculation.Calculation
	if len(calcs) > 0 {
		if shown, err = s.shown(calcs, allowedIn(snap)); err != nil {
			return RetrieveResponse{}, err
		}
	}
	res := RetrieveResponse{Channels: s.translateForward(resChannels, calcs, shown)}
	if req.IncludeStatus {
		// A hidden calculation shows no status: its status names what it reads.
		owners := statusOwners(resChannels, shown)
		statuses, err := s.status.Join[struct{}](ctx, snap, owners)
		if err != nil {
			return RetrieveResponse{}, err
		}
		for i, st := range statuses {
			// A v0.62 client knows no calculation owner, so the old API names the
			// channel.
			if st != nil {
				st.Owner = new(channel.ResourceID(resChannels[i].Key()))
			}
			res.Channels[i].Status = st
		}
	}
	if req.IncludeUses {
		if err = s.fillUses(ctx, snap, res.Channels); err != nil {
			return RetrieveResponse{}, err
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
	// The rewrite of the virtual line moves a top-level name under an and.
	named := namedExactly(req.Where)
	if req.Where != nil && hasVirtual(*req.Where) {
		outputs, err := s.outputs(ctx, snap)
		if err != nil {
			return nil, err
		}
		where := withOldVirtual(*req.Where, outputs)
		req.Where = &where
	}
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
	if !named {
		q = q.Where(bind.Filter(snap, channel.Match))
	}
	if err = q.Exec(ctx, snap.Snapshot); err != nil {
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

// outputs returns the output of every calculation, whether or not the subject of snap
// can retrieve the calculation. A channel that the subject cannot retrieve stays out of
// a retrieve through the channel access filter.
func (s *Service) outputs(
	ctx context.Context,
	snap bind.Snapshot,
) ([]channel.Key, error) {
	var calcs []calculation.Calculation
	if err := s.calculations.NewRetrieve().Entries(&calcs).Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.Map(calcs, func(c calculation.Calculation, _ int) channel.Key {
		return c.Output
	}), nil
}

// hasVirtual reports whether w or a child of w filters on virtual.
func hasVirtual(w channel.Where) bool {
	return w.Virtual != nil ||
		slices.ContainsFunc(w.And, hasVirtual) ||
		slices.ContainsFunc(w.Or, hasVirtual) ||
		(w.Not != nil && hasVirtual(*w.Not))
}

// withOldVirtual returns w with the meaning of virtual that the old channel API keeps:
// a calculation's output is not virtual. outputs are the outputs of every calculation.
func withOldVirtual(w channel.Where, outputs []channel.Key) channel.Where {
	// An empty Or matches nothing, so a nil child list stays nil.
	if w.And != nil {
		w.And = lo.Map(w.And, func(c channel.Where, _ int) channel.Where {
			return withOldVirtual(c, outputs)
		})
	}
	if w.Or != nil {
		w.Or = lo.Map(w.Or, func(c channel.Where, _ int) channel.Where {
			return withOldVirtual(c, outputs)
		})
	}
	if w.Not != nil {
		not := withOldVirtual(*w.Not, outputs)
		w.Not = &not
	}
	if w.Virtual == nil || len(outputs) == 0 {
		return w
	}
	output := channel.Where{Key: &filter.Set[channel.Key]{In: outputs}}
	virtual := channel.Where{And: []channel.Where{
		{Virtual: new(true)},
		{Not: &output},
	}}
	rest := w
	rest.Virtual = nil
	if *w.Virtual {
		return channel.Where{And: []channel.Where{rest, virtual}}
	}
	return channel.Where{And: []channel.Where{rest, {Not: &virtual}}}
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

// translateForward returns the old API form of channels. Each channel that a
// calculation in calcs writes is calculated, and has the expression and operations of
// that calculation when shown holds it.
func (s *Service) translateForward(
	channels []channel.Channel,
	calcs map[channel.Key]calculation.Calculation,
	shown map[channel.Key]calculation.Calculation,
) []Channel {
	translated := translateChannelsForward(channels, s.internal.IsBuiltIn)
	for i := range translated {
		_, translated[i].Calculated = calcs[translated[i].Key]
		c, ok := shown[translated[i].Key]
		if !ok {
			continue
		}
		translated[i].Expression = c.Expression
		translated[i].Operations = calculation.ToChannelOperations(c.Operations)
	}
	return translated
}

// virtualKeys returns the keys of the virtual channels of channels, the only channels
// that a calculation can write.
func virtualKeys(channels []channel.Channel) channel.Keys {
	n := 0
	for _, ch := range channels {
		if ch.Virtual {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	keys := make(channel.Keys, 0, n)
	for _, ch := range channels {
		if ch.Virtual {
			keys = append(keys, ch.Key())
		}
	}
	return keys
}

// shown returns the calculations of calcs that readable allows, given the record and
// every channel it links. The old API shows the output of any other calculation as a
// plain virtual channel.
func (s *Service) shown(
	calcs map[channel.Key]calculation.Calculation,
	readable func(ids []resource.ID) (bool, error),
) (map[channel.Key]calculation.Calculation, error) {
	if len(calcs) == 0 {
		return nil, nil
	}
	shown := make(map[channel.Key]calculation.Calculation, len(calcs))
	for output, c := range calcs {
		ids := append(
			[]resource.ID{calculation.ResourceID(c.Key)},
			s.calculations.Channels(c)...,
		)
		ok, err := readable(ids)
		if err != nil {
			return nil, err
		}
		if ok {
			shown[output] = c
		}
	}
	return shown, nil
}

// allowedIn returns a check of whether the subject of snap can retrieve every one of
// ids.
func allowedIn(snap bind.Snapshot) func(ids []resource.ID) (bool, error) {
	return func(ids []resource.ID) (bool, error) {
		return !slices.ContainsFunc(ids, func(id resource.ID) bool {
			return !snap.Allowed(id)
		}), nil
	}
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
			Concurrency: ch.Concurrency,
			Internal:    builtIn(ch.Key()),
			Group:       ch.Group,
			Description: ch.Description,
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
			Group:       ch.Group,
			Description: ch.Description,
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
// internal channel, or a name that several channels hold, is a validation error.
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

// editedInputs returns the inputs of the calculation that writes the channel with key,
// which an edit compiled in tx, so its stored reads are current.
func (s *Service) editedInputs(
	ctx context.Context,
	tx gorp.Tx,
	key channel.Key,
) (channel.Keys, error) {
	calcs, err := s.calculations.Outputs(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	c := calcs[key]
	return s.calculations.Inputs(ctx, tx, nil, []calculation.Unsaved{{
		Output:     channel.Channel{Leaseholder: key.Lease(), LocalKey: key.LocalKey()},
		Expression: c.Expression,
		Operations: c.Operations,
		Compiled:   true,
		Reads:      c.Reads,
	}})
}

// Dispatch applies the action sequence to the target channel atomically. Subscribers to
// the channel action channel receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (DispatchResponse, error) {
	computed, err := s.applyChannel(ctx, tx, req.DispatchKey, req.Key, req.Actions)
	if err != nil {
		return DispatchResponse{}, s.describe(ctx, tx, err)
	}
	if !slices.ContainsFunc(req.Actions, editsCalculation) {
		return DispatchResponse{Actions: computed}, nil
	}
	inputs, err := s.editedInputs(ctx, tx, req.Key)
	if err != nil {
		return DispatchResponse{}, s.describe(ctx, tx, err)
	}
	if err = s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: framer.ResourceIDs(inputs),
	}); err != nil {
		return DispatchResponse{}, err
	}
	return DispatchResponse{Actions: computed}, nil
}
