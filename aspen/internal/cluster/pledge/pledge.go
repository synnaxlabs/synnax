// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pledge provides a system for pledging a node to a jury of candidates. The
// pledge uses quorum consensus to assign the node a unique name.
//
// To pledge a new node to a jury, call Pledge with a set of peer addresses. To
// register a node as a candidate, use Arbitrate.
//
// Vocabulary:
//
//   - Pledge: Used as both a verb and a noun. A PledgeServer is a node that has pledged
//     itself to the cluster. Pledging is the entire process of contacting a peer,
//     proposing a name to a jury, and returning it to the pledging node.
//
//   - Responsible: A node responsible for coordinating the pledge process. The
//     responsible node is the first peer that accepts the pledge request from the
//     pledging node.
//
//   - Candidate: A node in the pool of nodes that can be selected to form a jury that
//     arbitrates a pledge.
//
//   - Jury: A quorum (numCandidates/2 + 1) of candidates that arbitrate a pledge. All
//     jurors must accept the pledge for the node to be inducted.
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

// errProposalRejected is an internal error returned when a juror rejects a pledge
// proposal from a responsible node.
var errProposalRejected = errors.New("proposal rejected")

// Pledge pledges a new node to the cluster. This node, called the pledge, submits a
// request for ID assignment to a peer in peers. If the cluster approves the request,
// the node will be assigned an name and registered to arbitrate in future pledge (see
// [Arbitrate] for more on how this works). Keys of nodes in the cluster are guaranteed
// to be unique, but are not guaranteed to be sequential. Pledge will continue to
// contact peers at a scaling interval (defined in [Config.RetryScale] and
// [Config.RetryInterval]) until the cluster approves request or the provided context is
// cancelled.
func Pledge(ctx context.Context, cfgs ...Config) (Response, error) {
	if ctx.Err() != nil {
		return Response{}, ctx.Err()
	}
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return Response{}, err
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

	var res Response
	for addr := range addresses {
		select {
		case <-ctx.Done():
			return Response{}, errors.Combine(ctx.Err(), err)
		case dur := <-t.C:
			cfg.L.Info("pledging to peer", zap.Stringer("address", addr))

			reqCtx, cancel := context.WithTimeout(
				context.Background(),
				cfg.RequestTimeout,
			)

			res, err = cfg.TransportClient.Send(reqCtx, addr, Request{Key: 0})

			cancel()
			if err == nil {
				cfg.L.Info(
					"pledge successful",
					zap.Uint32("assigned_host", uint32(res.Key)),
					zap.Stringer("cluster_key", res.ClusterKey),
				)

				// If the pledge node has been inducted successfully, allow it to
				// arbitrate in future pledges.
				cfg.ClusterKey = res.ClusterKey
				arbitrate(cfg)
				return res, nil
			}
			if ctx.Err() != nil {
				return Response{}, errors.Combine(ctx.Err(), err)
			}
			cfg.L.Warn(
				"failed to pledge, retrying",
				zap.Duration("next_retry", dur),
				zap.Error(err),
			)
		}
	}
	if err != nil {
		return Response{}, err
	}
	return res, nil
}

// Arbitrate registers a node to arbitrate future pledges. When processing a pledge
// request, the node will act as responsible for the pledge and submit proposed keys to
// a jury of candidate nodes. The responsible node will continue to propose keys until
// [Config.MaxProposals] is reached. When processing a responsible node's proposal, the
// node will act a juror, and decide if it approves of the proposed name or not.
func Arbitrate(cfgs ...Config) error {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return err
	}
	cfg.R.Prod("pledge", cfg)
	cfg.L.Debug("registering node as pledge arbitrator", cfg.Report().ZapFields()...)
	arbitrate(cfg)
	return nil
}

func arbitrate(cfg Config) {
	j := &juror{Config: cfg}
	cfg.TransportServer.BindHandler(
		func(ctx context.Context, req Request) (Response, error) {
			if req.Key == 0 {
				return (&responsible{Config: cfg}).propose(ctx)
			}
			return Response{}, j.verdict(ctx, req)
		},
	)
}

type responsible struct {
	candidateSnapshot node.Group
	Config
	proposedKey node.Key
}

func (r *responsible) propose(ctx context.Context) (res Response, err error) {
	r.L.Info("responsible received pledge. starting proposal process.")

	ctx, span := r.T.Prod(ctx, "responsible.propose")
	defer func() { err = span.EndWith(err) }()

	res.ClusterKey = r.ClusterKey

	var propC uint
	for propC < r.MaxProposals {
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = errors.Combine(err, ctxErr)
			break
		}

		// pull in the latest candidateSnapshot. We manually refresh Candidates to
		// provide a consistent view through the lifetime of the proposal.
		r.refreshCandidates()

		// Add the proposed name unconditionally. Quorum juror's store each approved
		// request. If one node in the quorum is unreachable, other Candidates may have
		// already approved the request. This means that if we retry the request without
		// incrementing the proposed name, we'll get a rejection from the candidate that
		// approved the request last time. This will result in marginally higher keys
		// being assigned, but it's better than adding a lot of extra logic to the
		// proposal process.
		res.Key = r.idToPropose()

		quorum, qErr := r.buildQuorum()
		if qErr != nil {
			err = qErr
			break
		}

		logKey := zap.Uint32("key", uint32(res.Key))
		r.L.Debug("responsible proposing", logKey, zap.Int("quorumCount", len(quorum)))

		// If any node returns an error, it means we need to retry the responsible with
		// a new Name.
		if err = r.consultQuorum(ctx, res.Key, quorum); err != nil {
			// A rejection means the key is stale, and the retry proposes a higher one,
			// so it makes progress. Only infrastructure failures count against
			// MaxProposals: otherwise keys approved during failed proposals force every
			// retry to burn its budget re-traversing them, a livelock.
			if errors.Is(err, errProposalRejected) {
				r.L.Debug("quorum rejected proposal. retrying.", zap.Error(err))
			} else {
				propC++
				r.L.Error("quorum consultation failed. retrying.", zap.Error(err))
			}
			continue
		}

		r.L.Debug("quorum accepted pledge", logKey)

		// If no candidate returned an error, it means we reached a quorum approval, and
		// we can safely return the new Name to the caller.
		return res, nil
	}
	r.L.Error(
		"responsible failed to build healthy quorum",
		zap.Uint("numProposals", propC),
		zap.Error(err),
	)
	return Response{}, err
}

func (r *responsible) refreshCandidates() { r.candidateSnapshot = r.Candidates() }

func (r *responsible) buildQuorum() (node.Group, error) {
	presentCandidates := r.candidateSnapshot.WhereActive()
	size := len(presentCandidates)/2 + 1
	healthy := presentCandidates.WhereState(node.StateHealthy)
	if len(healthy) < size {
		return node.Group{}, errors.New("quorum unreachable")
	}
	return xrand.SubMap(healthy, size), nil
}

func (r *responsible) idToPropose() node.Key {
	if r.proposedKey == 0 {
		r.proposedKey = highestNodeID(r.candidateSnapshot) + 1
	} else {
		r.proposedKey++
	}
	return r.proposedKey
}

func (r *responsible) consultQuorum(
	ctx context.Context,
	key node.Key,
	quorum node.Group,
) error {
	reqCtx, cancel := context.WithTimeout(ctx, r.RequestTimeout)
	defer cancel()
	var (
		mu       sync.Mutex
		infraErr error
		rejected bool
	)
	wg := errgroup.Group{}
	for _, n := range quorum {
		wg.Go(func() error {
			_, err := r.TransportClient.Send(reqCtx, n.Address, Request{Key: key})
			if err == nil {
				return nil
			}
			// If any node returns an error, we need to retry the entire responsible, so
			// we need to cancel all running requests.
			defer cancel()
			mu.Lock()
			defer mu.Unlock()
			if errors.Is(err, errProposalRejected) {
				rejected = true
				r.L.Debug(
					"quorum rejected proposal",
					zap.Uint32("key", uint32(key)),
					zap.Stringer("address", n.Address),
				)
			} else if !errors.Is(err, context.Canceled) {
				infraErr = err
				r.L.Error(
					"failed to reach juror",
					zap.Uint32("key", uint32(key)),
					zap.Stringer("address", n.Address),
				)
			}
			return err
		})
	}
	err := wg.Wait()
	// The first error to cancel the round makes the sibling requests fail with
	// context.Canceled, so the error errgroup surfaces is racy. Report why the round
	// actually failed: a real infrastructure error, then a rejection, so that
	// MaxProposals accounting sees mixed rounds deterministically.
	if infraErr != nil {
		return infraErr
	}
	if rejected {
		return errProposalRejected
	}
	return err
}

type juror struct {
	approvals []node.Key
	Config
	mu sync.Mutex
}

func (j *juror) verdict(ctx context.Context, req Request) (err error) {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, span := j.T.Prod(ctx, "juror.verdict")
	defer func() { err = span.EndWith(err, errProposalRejected) }()
	logID := zap.Uint32("key", uint32(req.Key))
	j.L.Debug("juror received proposal. making verdict", logID)
	j.mu.Lock()
	defer j.mu.Unlock()
	if slices.Contains(j.approvals, req.Key) {
		j.L.Warn(
			"juror rejected proposal. already approved for a different pledge",
			logID,
		)
		return errProposalRejected
	}
	if req.Key <= highestNodeID(j.Candidates()) {
		j.L.Warn("juror rejected proposal. id out of range", logID)
		return errProposalRejected
	}
	j.approvals = append(j.approvals, req.Key)
	j.L.Debug("juror approved proposal", logID)
	return nil
}

func highestNodeID(candidates node.Group) node.Key {
	return lo.Max(lo.Keys(candidates))
}

func introduceRandomJitter(retryInterval time.Duration) {
	// sleep for a random percentage of the retry interval, somewhere between 0 and 25%.
	t := rand.Intn(int(retryInterval / 4))
	time.Sleep(time.Duration(t))
}
