// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pledge provides a system for pledging a node to a jury of Candidates.
// The pledge uses quorum consensus to assign the node a unique Name.
//
// To pledge a new node to a jury, call Pledge() with a set of peer addresses.
// To register a node as a candidate, use Arbitrate().
//
// Vocabulary:
//
//	Pledge - Used as both a verb and noun. A "PledgeServer" is a node that has
//	'pledged' itself to the cluster. 'Pledging' is the entire process of
//	contacting a peer, proposing an Name to a jury, and returning it to the pledge.
//	Responsible - A node that is responsible for coordinating the Pledge process.
//	A responsible node is the first peer that accepts the Pledge request from
//	the pledge node.
//	Candidates - A pool of nodes that can be selected to form a jury that can
//	arbitrate a Pledge.
//	Jury - A quorum (numCandidates/2 + 1) of Candidates that arbitrate a
//	Pledge. All jurors must accept the Pledge for the node to be inducted.
//
// RFC-2 provides details on how the pledging algorithm is implemented.
package pledge

import (
	"context"
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xrand "github.com/synnaxlabs/x/rand"
	xslices "github.com/synnaxlabs/x/slices"
	xtime "github.com/synnaxlabs/x/time"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var (
	// errQuorumUnreachable is returned when a quorum jury cannot be safely assembled.
	errQuorumUnreachable = errors.New("quorum unreachable")
	// errProposalRejected is an internal error returned when a juror rejects a pledge
	// proposal from a responsible node.
	errProposalRejected = errors.New("proposal rejected")
)

// Pledge pledges a new node to the cluster. This node, called the Pledge,
// submits a request for id assignment to a peer in peers. If the cluster approves
// the request, the node will be assigned an Name and registered to arbitrate in
// future pledge (see the Arbitrate function for more on how this works). keys
// of nodes in the cluster are guaranteed to be unique, but are not guaranteed
// to be sequential. Pledge will continue to contact peers at a scaling interval
// (defined in cfg.RetryScale and cfg.RetryInterval) until the cluster approves
// request or the provided context is cancelled. To see the required configuration
// parameters, see the Config struct.
func Pledge(ctx context.Context, cfgs ...Config) (res Response, err error) {
	if ctx.Err() != nil {
		return res, ctx.Err()
	}
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return res, err
	}

	cfg.R.Prod("pledge", cfg)
	cfg.L.Debug("beginning pledge process", cfg.Report().ZapFields()...)
	ctx, tracer := cfg.T.Prod(ctx, "pledge")
	defer tracer.End()

	// introduce random jitter to avoid a thundering herd during concurrent pledging.
	introduceRandomJitter(cfg.RetryInterval)

	addresses := xslices.IterEndlessly(cfg.Peers)

	t := xtime.NewScaledTicker(cfg.RetryInterval, cfg.RetryScale)
	defer t.Stop()

	for addr := range addresses {
		select {
		case <-ctx.Done():
			return Response{}, errors.Combine(ctx.Err(), err)
		case dur := <-t.C:
			cfg.L.Info("pledging to peer", zap.Stringer("address", addr))

			reqCtx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)

			res, err = cfg.TransportClient.Send(reqCtx, addr, Request{Key: 0})

			cancel()
			if err == nil {
				cfg.L.Info(
					"pledge successful",
					zap.Uint32("assignedHost", uint32(res.Key)),
					zap.Stringer("clusterKey", res.ClusterKey),
				)

				// If the pledge node has been inducted successfully, allow it to
				// arbitrate in future pledges.
				cfg.ClusterKey = res.ClusterKey
				return res, arbitrate(cfg)
			}
			if ctx.Err() != nil {
				return res, errors.Combine(ctx.Err(), err)
			}
			cfg.L.Warn("failed to pledge, retrying",
				zap.Duration("nextRetry", dur),
				zap.Error(err),
			)
		}
	}
	return res, err
}

// Arbitrate registers a node to arbitrate future pledges. When processing a pledge
// request, the node will act as responsible for the pledge and submit proposed
// keys to a jury of candidate nodes. The responsible node will continue to propose
// keys until cfg.MaxProposals is reached. When processing a responisble's proposal,
// the node will act a juror, and decide if it approves of the proposed Name
// or not. To see the required configuration parameters, see the Config struct.
func Arbitrate(cfgs ...Config) error {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return err
	}
	cfg.R.Prod("pledge", cfg)
	cfg.L.Debug("registering node as pledge arbitrator", cfg.Report().ZapFields()...)
	return arbitrate(cfg)
}

func arbitrate(cfg Config) error {
	j := &juror{Config: cfg}
	cfg.TransportServer.BindHandler(func(ctx context.Context, req Request) (Response, error) {
		if req.Key == 0 {
			return (&responsible{Config: cfg}).propose(ctx)
		}
		return Response{}, j.verdict(ctx, req)
	})
	return nil
}

type responsible struct {
	Config
	candidateSnapshot node.Group
	_proposedKey      node.Key
}

func (r *responsible) propose(ctx context.Context) (res Response, err error) {
	r.L.Info("responsible received pledge. starting proposal process.")

	ctx, span := r.T.Prod(ctx, "responsible.propose")
	defer func() { _ = span.EndWith(err) }()

	res.ClusterKey = r.ClusterKey

	var propC int
	for propC = range r.MaxProposals {
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = errors.Combine(err, ctxErr)
			break
		}

		// pull in the latest candidateSnapshot. We manually refresh Candidates
		// to provide a consistent view through the lifetime of the proposal.
		r.refreshCandidates()

		// Add the proposed Name unconditionally. Quorum juror's store each
		// approved request. If one node in the quorum is unreachable, other
		// Candidates may have already approved the request. This means that
		// if we retry the request without incrementing the proposed Name, we'll
		// get a rejection from the candidate that approved the request last time.
		// This will result in marginally higher keys being assigned, but it's
		// better than adding a lot of extra logic to the proposal process.
		res.Key = r.idToPropose()

		quorum, qErr := r.buildQuorum()
		if qErr != nil {
			err = qErr
			break
		}

		logKey := zap.Uint32("key", uint32(res.Key))
		r.L.Debug("responsible proposing", logKey, zap.Int("quorumCount", len(quorum)))

		// If any node returns an error, it means we need to retry the responsible with a new Name.
		if err = r.consultQuorum(ctx, res.Key, quorum); err != nil {
			r.L.Error("quorum rejected proposal. retrying.", zap.Error(err))
			continue
		}

		r.L.Debug("quorum accepted pledge", logKey)

		// If no candidate returned an error, it means we reached a quorum approval,
		// and we can safely return the new Name to the caller.
		return res, nil
	}
	r.L.Error(
		"responsible failed to build healthy quorum",
		zap.Int("numProposals", propC),
		zap.Error(err),
	)
	return res, err
}

func (r *responsible) refreshCandidates() { r.candidateSnapshot = r.Candidates() }

func (r *responsible) buildQuorum() (node.Group, error) {
	presentCandidates := r.candidateSnapshot.WhereActive()
	size := len(presentCandidates)/2 + 1
	healthy := presentCandidates.WhereState(node.StateHealthy)
	if len(healthy) < size {
		return node.Group{}, errQuorumUnreachable
	}
	return xrand.SubMap(healthy, size), nil
}

func (r *responsible) idToPropose() node.Key {
	if r._proposedKey == 0 {
		r._proposedKey = highestNodeID(r.candidateSnapshot) + 1
	} else {
		r._proposedKey++
	}
	return r._proposedKey
}

func (r *responsible) consultQuorum(ctx context.Context, key node.Key, quorum node.Group) error {
	reqCtx, cancel := context.WithTimeout(ctx, r.RequestTimeout)
	defer cancel()
	wg := errgroup.Group{}
	for _, n := range quorum {
		n_ := n
		wg.Go(func() error {
			_, err := r.TransportClient.Send(reqCtx, n_.Address, Request{Key: key})
			if errors.Is(err, errProposalRejected) {
				r.L.Debug(
					"quorum rejected proposal",
					zap.Uint32("key", uint32(key)),
					zap.Stringer("address", n_.Address),
				)
				cancel()
			}
			// If any node returns an error, we need to retry the entire responsible,
			// so we need to cancel all running requests.
			if err != nil {
				r.L.Error("failed to reach juror",
					zap.Uint32("key", uint32(key)),
					zap.Stringer("address", n_.Address),
				)
				cancel()
			}
			return err
		})
	}
	return wg.Wait()
}

type juror struct {
	Config
	mu        sync.Mutex
	approvals []node.Key
}

func (j *juror) verdict(ctx context.Context, req Request) (err error) {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, span := j.T.Prod(ctx, "juror.verdict")
	defer func() { _ = span.EndWith(err, errProposalRejected) }()
	logID := zap.Uint32("key", uint32(req.Key))
	j.L.Debug("juror received proposal. making verdict", logID)
	j.mu.Lock()
	defer j.mu.Unlock()
	if slices.Contains(j.approvals, req.Key) {
		j.L.Warn("juror rejected proposal. already approved for a different pledge", logID)
		err = errProposalRejected
		return
	}
	if req.Key <= highestNodeID(j.Candidates()) {
		j.L.Warn("juror rejected proposal. id out of range", logID)
		err = errProposalRejected
	}
	j.approvals = append(j.approvals, req.Key)
	j.L.Debug("juror approved proposal", logID)
	return
}

func highestNodeID(candidates node.Group) node.Key { return lo.Max(lo.Keys(candidates)) }

func introduceRandomJitter(retryInterval time.Duration) {
	// sleep for a random percentage of the retry interval, somewhere between
	// 0 and 25%.
	t := rand.Intn(int(retryInterval / 4))
	time.Sleep(time.Duration(t))
}
