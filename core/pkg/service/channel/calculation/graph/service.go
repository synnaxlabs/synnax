// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with this file, use of
// this software will be governed by the Apache License, Version 2.0, included in the
// file licenses/APL.txt.

package graph

import (
	"context"
	"sync"

	channel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/symbol"
	channelanalyzer "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	calcompiler "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

type StatusDetails struct {
	Channel channel.Key `json:"channel" msgpack:"channel"`
}

type node struct {
	channel.Channel
	deps       channel.Keys
	unresolved []string
	invalid    bool
}

type Service struct {
	distribution *channel.Service
	status       status.Writer[StatusDetails]
	disconnect   observe.Disconnect
	mu           struct {
		nodes            map[channel.Key]node
		dependents       map[channel.Key]map[channel.Key]struct{}
		unresolvedByName map[string]map[channel.Key]struct{}
		sync.RWMutex
	}
}

func OpenService(
	ctx context.Context,
	distribution *channel.Service,
	statusSvc *status.Service,
) (*Service, error) {
	s := &Service{
		distribution: distribution,
		status:       status.NewWriter[StatusDetails](statusSvc, nil),
	}
	s.mu.nodes = make(map[channel.Key]node)
	s.mu.dependents = make(map[channel.Key]map[channel.Key]struct{})
	s.mu.unresolvedByName = make(map[string]map[channel.Key]struct{})
	if err := s.hydrate(ctx); err != nil {
		return nil, err
	}
	s.disconnect = distribution.NewObservable().OnChange(s.handleChanges)
	return s, nil
}

func (s *Service) Close() error {
	if s.disconnect != nil {
		s.disconnect()
	}
	return nil
}

func (s *Service) hydrate(ctx context.Context) error {
	var channels []channel.Channel
	if err := s.distribution.NewRetrieve().Entries(&channels).Exec(ctx, nil); err != nil {
		return err
	}
	repairs := make([]channel.Channel, 0)
	for {
		changed := false
		nextNodes := make(map[channel.Key]node)
		nextDependents := make(map[channel.Key]map[channel.Key]struct{})
		nextUnresolved := make(map[string]map[channel.Key]struct{})
		for _, ch := range channels {
			if !ch.IsCalculated() {
				continue
			}
			nd, err := s.inspectNode(ctx, nil, ch, nil)
			if err != nil {
				s.setNodeStatus(ctx, ch.Key(), ch.Name, err)
			} else {
				_ = s.clearNodeStatus(ctx, ch.Key())
			}
			upsertNode(nextNodes, nextDependents, nextUnresolved, nd)
			if !nd.invalid && ch.DataType != nd.DataType {
				ch.DataType = nd.DataType
				repairs = append(repairs, ch)
				changed = true
			}
		}
		if !changed {
			s.mu.Lock()
			s.mu.nodes = nextNodes
			s.mu.dependents = nextDependents
			s.mu.unresolvedByName = nextUnresolved
			s.mu.Unlock()
			break
		}
	}
	if len(repairs) > 0 {
		if err := s.distribution.NewWriter(nil).CreateMany(ctx, &repairs); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) handleChanges(ctx context.Context, reader gorp.TxReader[channel.Key, channel.Channel]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	queued := make(map[channel.Key]struct{})
	var unresolvedNames []string
	for chg := range reader {
		ch := chg.Value
		if chg.Variant == change.VariantDelete {
			s.removeNode(chg.Key)
			if ch.Name != "" {
				unresolvedNames = append(unresolvedNames, ch.Name)
			}
			s.enqueueDependents(chg.Key, queued)
			continue
		}
		if ch.IsCalculated() {
			node, err := s.inspectNode(ctx, nil, ch, nil)
			if err != nil {
				s.setNodeStatus(ctx, ch.Key(), ch.Name, err)
			} else {
				_ = s.clearNodeStatus(ctx, ch.Key())
			}
			s.upsertNode(node)
			s.enqueueDependents(ch.Key(), queued)
			continue
		}
		s.enqueueDependents(ch.Key(), queued)
		unresolvedNames = append(unresolvedNames, ch.Name)
	}
	_ = s.reconcileQueued(ctx, nil, queued, unresolvedNames, nil, false)
}

func (s *Service) setNodeStatus(ctx context.Context, key channel.Key, name string, err error) {
	_ = s.status.Set(ctx, &status.Status[StatusDetails]{
		Key:         channel.OntologyID(key).String(),
		Name:        name,
		Variant:     xstatus.VariantError,
		Message:     "failed to analyze calculated channel",
		Description: err.Error(),
		Time:        telem.Now(),
		Details:     StatusDetails{Channel: key},
	})
}

func (s *Service) clearNodeStatus(ctx context.Context, key channel.Key) error {
	return s.status.Delete(ctx, channel.OntologyID(key).String())
}

func (s *Service) inspectNode(
	ctx context.Context,
	tx gorp.Tx,
	ch channel.Channel,
	overlayMap map[channel.Key]channel.Channel,
) (node, error) {
	dt, err := channelanalyzer.New(symbol.NewResolver(s.distribution, tx)).
		Analyze(ctx, ch.Name, ch.Expression)
	nd := node{Channel: ch}
	if ch.Key() == 0 {
		nd.LocalKey = 1
	}
	if err == nil {
		nd.DataType = dt
		prog, preErr := calcompiler.PreProcess(ctx, calcompiler.Config{
			ChannelService: s.distribution,
			Channel:        nd.Channel,
			SymbolResolver: symbol.NewResolver(s.distribution, tx),
		})
		if preErr != nil {
			err = preErr
		} else if len(prog.Functions) > 0 {
			nd.deps = make(channel.Keys, 0, len(prog.Functions[0].Channels.Read))
			for key := range prog.Functions[0].Channels.Read {
				nd.deps = append(nd.deps, channel.Key(key))
			}
		}
	}
	nd.invalid = err != nil
	return nd, err
}

func (s *Service) reconcileQueued(
	ctx context.Context,
	tx gorp.Tx,
	queued map[channel.Key]struct{},
	unresolvedNames []string,
	overlayMap map[channel.Key]channel.Channel,
	persist bool,
) error {
	if overlayMap == nil {
		overlayMap = make(map[channel.Key]channel.Channel)
	}
	s.enqueueUnresolved(unresolvedNames, queued)
	updates := make([]channel.Channel, 0)
	for len(queued) > 0 {
		next := make(map[channel.Key]struct{})
		for key := range queued {
			nd, ok := s.mu.nodes[key]
			if !ok {
				continue
			}
			refetched := nd.Channel
			if err := s.distribution.NewRetrieve().WhereKeys(key).Entry(&refetched).Exec(ctx, tx); err != nil {
				continue
			}
			newNode, err := s.inspectNode(ctx, tx, refetched, overlayMap)
			oldInvalid := nd.invalid
			oldType := nd.DataType
			s.upsertNode(newNode)
			if err != nil {
				s.setNodeStatus(ctx, key, refetched.Name, err)
				continue
			}
			_ = s.clearNodeStatus(ctx, key)
			if oldInvalid || oldType != newNode.DataType {
				if persist && oldType != newNode.DataType {
					updates = append(updates, newNode.Channel)
					overlayMap[key] = newNode.Channel
				}
				s.enqueueDependents(key, next)
			}
		}
		queued = next
	}
	if persist && len(updates) > 0 {
		if err := s.distribution.NewWriter(tx).CreateMany(ctx, &updates); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) removeNode(key channel.Key) {
	nd, ok := s.mu.nodes[key]
	if !ok {
		return
	}
	for _, dep := range nd.deps {
		delete(s.mu.dependents[dep], key)
		if len(s.mu.dependents[dep]) == 0 {
			delete(s.mu.dependents, dep)
		}
	}
	for _, name := range nd.unresolved {
		delete(s.mu.unresolvedByName[name], key)
		if len(s.mu.unresolvedByName[name]) == 0 {
			delete(s.mu.unresolvedByName, name)
		}
	}
	delete(s.mu.nodes, key)
}

func (s *Service) upsertNode(node node) {
	s.removeNode(node.Key())
	upsertNode(s.mu.nodes, s.mu.dependents, s.mu.unresolvedByName, node)
}

func upsertNode(
	nodes map[channel.Key]node,
	dependents map[channel.Key]map[channel.Key]struct{},
	unresolvedByName map[string]map[channel.Key]struct{},
	nd node,
) {
	nodes[nd.Key()] = nd
	for _, dep := range nd.deps {
		if dependents[dep] == nil {
			dependents[dep] = make(map[channel.Key]struct{})
		}
		dependents[dep][nd.Key()] = struct{}{}
	}
	for _, name := range nd.unresolved {
		if unresolvedByName[name] == nil {
			unresolvedByName[name] = make(map[channel.Key]struct{})
		}
		unresolvedByName[name][nd.Key()] = struct{}{}
	}
}

func (s *Service) enqueueDependents(key channel.Key, queued map[channel.Key]struct{}) {
	for dep := range s.mu.dependents[key] {
		queued[dep] = struct{}{}
	}
}

func (s *Service) enqueueUnresolved(names []string, queued map[channel.Key]struct{}) {
	for _, name := range names {
		for key := range s.mu.unresolvedByName[name] {
			queued[key] = struct{}{}
		}
	}
}
