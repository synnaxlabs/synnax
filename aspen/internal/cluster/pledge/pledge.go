// Package pledge provides a system for pledging a node to a jury of Candidates.
// The pledge uses quorum consensus to assign the node a unique ID.
//
// To pledge a new node to a jury, call Pledge() with a set of peer addresses.
// To register a node as a candidate, use Arbitrate().
//
// Vocabulary:
//
//	Pledge - Used as both a verb and noun. LocalKey "PledgeServer" is a node that has
//	'pledged' itself to the cluster. 'Pledging' is the entire process of
//	contacting a peer, proposing an ID to a jury, and returning it to the pledge.
//	Responsible - LocalKey node that is responsible for coordinating the Pledge process.
//	LocalKey responsible node is the first peer that accepts the Pledge request from
//	the pledge node.
//	Candidates - LocalKey pool of nodes that can be selected to form a jury that can
//	arbitrate a Pledge.
//	Jury - LocalKey quorum (numCandidates/2 + 1) of Candidates that arbitrate a
//	Pledge. All jurors must accept the Pledge for the node to be inducted.
//
// RFC-2 provides details on how the pledging algorithm is implemented.
package pledge

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/iter"
	xrand "github.com/synnaxlabs/x/rand"
	xtime "github.com/synnaxlabs/x/time"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"sync"
	"time"
)

var (
	// errQuorumUnreachable is returned when a quorum jury cannot be safely assembled.
	errQuorumUnreachable = errors.New("quorum unreachable")
	// proposalRejected is an internal error returned when a juror rejects a pledge
	// proposal from a responsible node.
	proposalRejected = errors.New("proposal rejected")
)

// Pledge pledges a new node to the cluster. This node, called the Pledge,
// submits a request for id assignment to a peer in peers. If the cluster approves
// the request, the node will be assigned an ID and registered to arbitrate in
// future pledge (see the Arbitrate function for more on how this works). IDs
// of nodes in the cluster are guaranteed to be unique, but are not guaranteed
// to be sequential. Pledge will continue to contact peers at a scaling interval
// (defined in cfg.RetryScale and cfg.RetryInterval) until the cluster approves
// request or the provided context is cancelled. To see the required configuration
// parameters, see the Config struct.
func Pledge(ctx context.Context, cfgs ...Config) (res Response, err error) {
	if ctx.Err() != nil {
		return res, ctx.Err()
	}
	cfg, err := config.OverrideAndValidate(DefaultConfig, cfgs...)
	if err != nil {
		return res, err
	}
	// Because peers are only required whe calling PledgeServer, we need to perform
	// this validation outside of config.Validate.
	if len(cfg.Peers) == 0 {
		return res, errors.New("[pledge] - at least one peer required")
	}

	alamos.AttachReporter(cfg.Experiment, "pledge", alamos.Debug, cfg)
	cfg.Logger.Infow("beginning pledge process", cfg.Report().LogArgs()...)

	// introduce random jitter to avoid a thundering herd during concurrent pledging.
	introduceRandomJitter(cfg.RetryInterval)

	nextAddr := iter.Endlessly(cfg.Peers)

	t := xtime.NewScaledTicker(cfg.RetryInterval, cfg.RetryScale)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return Response{}, errors.CombineErrors(ctx.Err(), err)
		case dur := <-t.C:
			addr := nextAddr()
			cfg.Logger.Infow("pledging to peer", "address", addr)
			reqCtx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
			res, err = cfg.TransportClient.Send(reqCtx, addr, Request{ID: 0})
			cancel()
			if err == nil {
				cfg.Logger.Infow(
					"pledge successful",
					"assignedHost", res.ID,
					"clusterKey", res.ClusterKey,
				)

				// If the pledge node has been inducted successfully, allow it to
				// arbitrate in future pledges.
				cfg.ClusterKey = res.ClusterKey
				return res, arbitrate(cfg)
			}
			if ctx.Err() != nil {
				return res, errors.CombineErrors(ctx.Err(), err)
			}
			cfg.Logger.Warnw("failed to pledge, retrying", "nextRetry", dur, "err", err)
		}
	}
}

// Arbitrate registers a node to arbitrate future pledges. When processing a pledge
// request, the node will act as responsible for the pledge and submit proposed
// IDs to a jury of candidate nodes. The responsible node will continue to propose
// IDs until cfg.MaxProposals is reached. When processing a responisble's proposal,
// the node will act a juror, and decide if it approves of the proposed ID
// or not. To see the required configuration parameters, see the Config struct.
func Arbitrate(cfgs ...Config) error {
	cfg, err := config.OverrideAndValidate(DefaultConfig, cfgs...)
	if err != nil {
		return err
	}
	alamos.AttachReporter(cfg.Experiment, "pledge", alamos.Debug, cfg)
	cfg.Logger.Infow("registering node as pledge arbitrator", cfg.Report().LogArgs()...)
	return arbitrate(cfg)
}

func arbitrate(cfg Config) error {
	j := &juror{Config: cfg}
	cfg.TransportServer.BindHandler(func(ctx context.Context, req Request) (Response, error) {
		if req.ID == 0 {
			return (&responsible{Config: cfg}).propose(ctx)
		}
		return Response{}, j.verdict(ctx, req)
	})
	return nil
}

type responsible struct {
	Config
	candidateSnapshot node.Group
	_proposedID       node.ID
}

func (r *responsible) propose(ctx context.Context) (res Response, err error) {
	r.Logger.Infow("responsible received pledge. starting proposal process.")

	res.ClusterKey = r.ClusterKey

	var propC int
	for propC = 0; propC < r.MaxProposals; propC++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = errors.CombineErrors(err, ctxErr)
			break
		}

		// pull in the latest candidateSnapshot. We manually refresh Candidates
		// to provide a consistent view through the lifetime of the proposal.
		r.refreshCandidates()

		// Add the proposed ID unconditionally. Quorum juror's store each
		// approved request. If one node in the quorum is unreachable, other
		// Candidates may have already approved the request. This means that
		// if we retry the request without incrementing the proposed ID, we'll
		// get a rejection from the candidate that approved the request last time.
		// This will result in marginally higher IDs being assigned, but it's
		// better than adding a lot of extra logic to the proposal process.
		res.ID = r.idToPropose()

		quorum, qErr := r.buildQuorum()
		if qErr != nil {
			err = qErr
			break
		}

		r.Logger.Debugw("responsible proposing", "id", res.ID, "quorumCount", len(quorum))

		// If any node returns an error, it means we need to retry the responsible with a new ID.
		if err = r.consultQuorum(ctx, res.ID, quorum); err != nil {
			r.Logger.Errorw("quorum rejected proposal. retrying.", "err", zap.Error(err))
			continue
		}

		r.Logger.Debugw("quorum accepted pledge", "id", res.ID)

		// If no candidate return an error, it means we reached a quorum approval,
		// and we can safely return the new ID to the caller.
		return res, nil
	}
	r.Logger.Errorw("responsible failed to build healthy quorum", "numProposals", propC, "err", err)
	return res, err
}

func (r *responsible) refreshCandidates() { r.candidateSnapshot = r.Config.Candidates() }

func (r *responsible) buildQuorum() (node.Group, error) {
	presentCandidates := r.candidateSnapshot.WhereActive()
	size := len(presentCandidates)/2 + 1
	healthy := presentCandidates.WhereState(node.StateHealthy)
	if len(healthy) < size {
		return node.Group{}, errQuorumUnreachable
	}
	return xrand.SubMap(healthy, size), nil
}

func (r *responsible) idToPropose() node.ID {
	if r._proposedID == 0 {
		r._proposedID = highestNodeID(r.candidateSnapshot) + 1
	} else {
		r._proposedID++
	}
	return r._proposedID
}

func (r *responsible) consultQuorum(ctx context.Context, id node.ID, quorum node.Group) error {
	reqCtx, cancel := context.WithTimeout(ctx, r.RequestTimeout)
	defer cancel()
	wg := errgroup.Group{}
	for _, n := range quorum {
		n_ := n
		wg.Go(func() error {
			_, err := r.TransportClient.Send(reqCtx, n_.Address, Request{ID: id})
			if errors.Is(err, proposalRejected) {
				r.Logger.Debugw("quorum rejected proposal", "id", id, "address", n_.Address)
				cancel()
			}
			// If any node returns an error, we need to retry the entire responsible,
			// so we need to cancel all running requests.
			if err != nil {
				r.Logger.Errorw("failed to reach juror", "id", id, "err", err)
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
	approvals []node.ID
}

func (j *juror) verdict(ctx context.Context, req Request) (err error) {
	j.Logger.Debugw("juror received proposal. making verdict", "id", req.ID)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, appID := range j.approvals {
		if appID == req.ID {
			j.Logger.Warnw("juror rejected proposal. already approved for a different pledge", "id", req.ID)
			return proposalRejected
		}
	}
	if req.ID <= highestNodeID(j.Candidates()) {
		j.Logger.Warnw("juror rejected proposal. id out of range", "id", req.ID)
		return proposalRejected
	}
	j.approvals = append(j.approvals, req.ID)
	j.Logger.Debugw("juror approved proposal", "id", req.ID)
	return nil
}

func highestNodeID(candidates node.Group) node.ID { return lo.Max(lo.Keys(candidates)) }

func introduceRandomJitter(retryInterval time.Duration) {
	// sleep for a random percentage of the retry interval, somewhere between
	// 0 and 25%.
	t := rand.Intn(int(retryInterval / 4))
	time.Sleep(time.Duration(t))
}
