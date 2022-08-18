// Package pledge provides a system for pledging a node to a jury of candidates.
// The pledge uses quorum consensus to assign the node a unique ID.
//
// To pledge a new node to a jury, call Pledge() with a set of peer addresses.
// To register a node as a candidate, use Arbitrate().
//
// Vocabulary:
//
//	Pledge - Used as both a verb and noun. A "Pledge" is a node that has
//	'pledged' itself to the cluster. 'Pledging' is the entire process of
//	contacting a peer, proposing an ID to a jury, and returning it to the pledge.
//	Responsible - A node that is responsible for coordinating the Pledge process.
//	A responsible node is the first peer that accepts the Pledge request from
//	the pledge node.
//	Candidates - A pool of nodes that can be selected to form a jury that can
//	arbitrate a Pledge.
//	Jury - A quorum (numCandidates/2 + 1) of candidates that arbitrate a
//	Pledge. All jurors must accept the Pledge for the node to be inducted.
//
// The following RFC provides details on how the pledging algorithm
// is implemented. https://github.com/arya-analytics/delta/blob/DA-153-aspen-rfc/docs/rfc/220518-aspen-p2p-network.md#adding-a-member.
package pledge

import (
	"context"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/iter"
	xrand "github.com/arya-analytics/x/rand"
	xtime "github.com/arya-analytics/x/time"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"sync"
	"time"
)

var (
	// errQuorumUnreachable is returned when a quorum jury cannot be safely assembled.
	errQuorumUnreachable = errors.New("quorum unreachable")
	// proposalRejected is an internal error returned when a juror rejects a pledge proposal from a responsible node.
	proposalRejected = errors.New("proposal rejected")
)

// Pledge pledges a node to a Jury selected from candidateSnapshot for membership.
// Pledge will submit a request to a peer in peers, this peer (called the responsible)
// will then issue a request to random quorum from candidates with a proposed ID.
// If the jury approves the request, the pledge will be given membership, assigned a
// unique ID, and allowed to Arbitrate in future proposals. See algorithm in
// package level documentation for implementation details. Although IDs are guaranteed
// to be unique, they are not guarantee to be sequential. Pledge will continue to
// contact peers in cfg.peerAddresses at a scaling interval until the provided
// context is cancelled.
func Pledge(
	ctx context.Context,
	peers []address.Address,
	candidates func() node.Group,
	cfg Config,
) (id node.ID, err error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	if len(peers) == 0 {
		return id, errors.New("[pledge] no peers provided")
	}

	cfg.peerAddresses, cfg.candidates = peers, candidates
	cfg = cfg.Merge(DefaultConfig())
	if err = cfg.Validate(); err != nil {
		return id, err
	}

	alamos.AttachReporter(cfg.Experiment, "pledge", alamos.Debug, cfg)
	cfg.Logger.Infow("beginning pledge process", cfg.LogArgs()...)

	// introduce random jitter to avoid a thundering herd during concurrent pledging.
	introduceRandomJitter(cfg.RetryInterval)

	nextAddr := iter.Endlessly(cfg.peerAddresses)

	t := xtime.NewScaledTicker(cfg.RetryInterval, cfg.RetryScale)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return 0, errors.CombineErrors(ctx.Err(), err)
		case dur := <-t.C:
			addr := nextAddr()
			cfg.Logger.Infow("pledging to peer", "address", addr)
			reqCtx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
			id, err = cfg.Transport.Send(reqCtx, addr, 0)
			cancel()
			if err == nil {
				cfg.Logger.Infow("pledge successful", "assignedHost", id)

				// If the pledge node has been inducted successfully, allow it to arbitrate
				// in future pledges.
				return id, arbitrate(cfg)
			}
			if ctx.Err() != nil {
				return 0, errors.CombineErrors(ctx.Err(), err)
			}
			cfg.Logger.Warnw("failed to pledge, retrying", "nextRetry", dur, "err", err)
		}
	}
}

// Arbitrate registers a node to arbitrate future pledges. When a node calls
// Arbitrate, it will be made available to become a Responsible or Juror node.
// Any node that calls arbitrate should also be a member of candidates.
func Arbitrate(candidates func() node.Group, cfg Config) error {
	cfg.candidates = candidates
	cfg = cfg.Merge(DefaultConfig())
	if err := cfg.Validate(); err != nil {
		return err
	}
	alamos.AttachReporter(cfg.Experiment, "pledge", alamos.Debug, cfg)
	cfg.Logger.Infow("registering node as pledge arbitrator", cfg.LogArgs()...)
	return arbitrate(cfg)
}

func arbitrate(cfg Config) error {
	j := &juror{Config: cfg}
	cfg.Transport.BindHandler(func(ctx context.Context, id node.ID) (node.ID, error) {
		if id == 0 {
			return (&responsible{Config: cfg}).propose(ctx)
		}
		return 0, j.verdict(ctx, id)
	})
	return nil
}

// |||||| RESPONSIBLE ||||||

type responsible struct {
	Config
	candidateSnapshot node.Group
	_proposedID       node.ID
}

func (r *responsible) propose(ctx context.Context) (id node.ID, err error) {
	r.Logger.Infow("responsible received pledge. starting proposal process.")

	var propC int
	for propC = 0; propC < r.MaxProposals; propC++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = errors.CombineErrors(err, ctxErr)
			break
		}

		// pull in the latest candidateSnapshot. We manually refresh candidates
		// to provide a consistent view through the lifetime of the proposal.
		r.refreshCandidates()

		// Increment the proposed ID unconditionally. Quorum juror's store each
		// approved request. If one node in the quorum is unreachable, other
		// candidates may have already approved the request. This means that
		// if we retry the request without incrementing the proposed ID, we'll
		// get a rejection from the candidate that approved the request last time.
		// This will result in marginally higher IDs being assigned, but it's
		// better than adding a lot of extra logic to the proposal process.
		id = r.idToPropose()

		quorum, qErr := r.buildQuorum()
		if qErr != nil {
			err = qErr
			break
		}

		r.Logger.Debugw("responsible proposing", "id", id, "quorumCount", len(quorum))

		// If any node returns an error, it means we need to retry the responsible with a new ID.
		if err = r.consultQuorum(ctx, id, quorum); err != nil {
			r.Logger.Errorw("quorum rejected proposal. retrying.", "err", zap.Error(err))
			continue
		}

		r.Logger.Debugw("quorum accepted pledge", "id", id)

		// If no candidate return an error, it means we reached a quorum approval,
		// and we can safely return the new ID to the caller.
		return id, nil
	}
	r.Logger.Errorw("responsible failed to build healthy quorum", "numProposals", propC, "err", err)
	return id, err
}

func (r *responsible) refreshCandidates() { r.candidateSnapshot = r.Config.candidates() }

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
			_, err := r.Transport.Send(reqCtx, n_.Address, id)
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

// |||||| JUROR ||||||

type juror struct {
	Config
	mu        sync.Mutex
	approvals []node.ID
}

func (j *juror) verdict(ctx context.Context, id node.ID) (err error) {
	j.Logger.Debugw("juror received proposal. making verdict", "id", id)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, appID := range j.approvals {
		if appID == id {
			j.Logger.Warnw("juror rejected proposal. already approved for a different pledge", "id", id)
			return proposalRejected
		}
	}
	if id <= highestNodeID(j.candidates()) {
		j.Logger.Warnw("juror rejected proposal. id out of range", "id", id)
		return proposalRejected
	}
	j.approvals = append(j.approvals, id)
	j.Logger.Debugw("juror approved proposal", "id", id)
	return nil
}

func highestNodeID(candidates node.Group) node.ID {
	return lo.Max(lo.Keys(candidates))
}

func introduceRandomJitter(retryInterval time.Duration) {
	// sleep for a random percentage of the retry interval, somewhere between
	// 0 and 25%.
	t := rand.Intn(int(retryInterval / 4))
	time.Sleep(time.Duration(t))
}
