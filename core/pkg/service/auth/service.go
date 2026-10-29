// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"
	"crypto/subtle"
	"io"
	"iter"
	"maps"
	"slices"
	"sync"
	"time"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// DB stores sessions.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Authenticators checks proofs, one per accepted method.
	//
	// [REQUIRED]
	Authenticators map[Method]Authenticator
	// IdleTimeout ends a session that no request has used for this long.
	//
	// [OPTIONAL] - Defaults to [DefaultIdleTimeout].
	IdleTimeout time.Duration
	// Clock tells the time, for the idle timeout.
	//
	// [OPTIONAL] - Defaults to the system clock.
	Clock xtime.Clock
	// HoldCheckInterval is how often held sessions are checked against storage, so a
	// revocation the change feed dropped still ends its streams.
	//
	// [OPTIONAL] - Defaults to 30 seconds.
	HoldCheckInterval time.Duration
	// Instrumentation is for logging, tracing, metrics, etc.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// DefaultIdleTimeout is the default [ServiceConfig.IdleTimeout].
const DefaultIdleTimeout = 7 * 24 * time.Hour

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Authenticators = override.Nil(c.Authenticators, other.Authenticators)
	c.IdleTimeout = override.Numeric(c.IdleTimeout, other.IdleTimeout)
	c.Clock = override.Nil(c.Clock, other.Clock)
	c.HoldCheckInterval = override.Numeric(c.HoldCheckInterval, other.HoldCheckInterval)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("auth")
	v.NotNil("db", c.DB)
	v.Ternary("authenticators", len(c.Authenticators) == 0, "must not be empty")
	v.Positive("idle_timeout", c.IdleTimeout)
	v.NotNil("clock", c.Clock)
	v.Positive("hold_check_interval", c.HoldCheckInterval)
	return v.Error()
}

// Service logs subjects in and checks the session tokens their requests carry.
type Service struct {
	cfg        ServiceConfig
	table      *gorp.Table[SessionKey, Session]
	indexes    indexes
	shutdown   io.Closer
	disconnect observe.Disconnect
	mu         struct {
		sync.Mutex
		holds map[SessionKey]*hold
		// seen leads the stored activity by up to a quarter of the idle timeout.
		seen map[SessionKey]telem.TimeStamp
	}
}

// hold counts the streams on this node that keep a session open.
type hold struct {
	count int
	ended chan struct{}
	over  bool
}

// OpenService opens a [Service] with the given configurations. It sweeps idle sessions
// in the background until Close.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(
		ServiceConfig{
			IdleTimeout:       DefaultIdleTimeout,
			Clock:             xtime.Real,
			HoldCheckInterval: 30 * time.Second,
		},
		cfgs...,
	)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, indexes: newIndexes()}
	s.mu.holds = make(map[SessionKey]*hold)
	s.mu.seen = make(map[SessionKey]telem.TimeStamp)
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[SessionKey, Session]{
		DB:              cfg.DB,
		Indexes:         s.indexes.all(),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	s.disconnect = s.table.Observe().OnChange(s.endDeletedHolds)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	s.shutdown = signal.NewHardShutdown(sCtx, cancel)
	signal.GoTick(
		sCtx,
		s.cfg.IdleTimeout/4,
		func(ctx context.Context, _ time.Time) error {
			// an error would stop the ticker
			if err := s.Sweep(ctx); err != nil {
				s.cfg.L.Error("failed to sweep idle sessions", zap.Error(err))
			}
			return nil
		},
		signal.WithRetryOnPanic(),
		signal.WithKey("session_sweep"),
	)
	signal.GoTick(
		sCtx,
		s.cfg.HoldCheckInterval,
		func(ctx context.Context, _ time.Time) error {
			if err := s.CheckHolds(ctx); err != nil {
				s.cfg.L.Error("failed to check held sessions", zap.Error(err))
			}
			return nil
		},
		signal.WithRetryOnPanic(),
		signal.WithKey("session_hold_check"),
	)
	return s, nil
}

// IdleTimeout returns how long a session lives without use.
func (s *Service) IdleTimeout() time.Duration { return s.cfg.IdleTimeout }

// Close stops the sweep and releases the service's resources.
func (s *Service) Close() error {
	s.disconnect()
	return errors.Combine(s.shutdown.Close(), s.table.Close())
}

// Hold marks the session as kept open by a stream on this node. A held session counts
// as active, so it never idles out, and every Sweep writes its activity for other
// nodes. The returned channel closes when the session ends, by revocation or by any
// node's sweep. The caller must call release when the stream closes. Hold returns
// [ErrInvalidToken] when no session has the key.
func (s *Service) Hold(
	ctx context.Context,
	key SessionKey,
) (ended <-chan struct{}, release func(), err error) {
	s.mu.Lock()
	h, ok := s.mu.holds[key]
	if !ok {
		h = &hold{ended: make(chan struct{})}
		s.mu.holds[key] = h
	}
	h.count++
	s.mu.Unlock()
	var once sync.Once
	release = func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if h.count--; h.count == 0 {
				delete(s.mu.holds, key)
			}
		})
	}
	// registered before the lookup, so a deletion in between still ends the hold
	exists, err := s.NewRetrieve().Where(MatchKeys(key)).Exists(ctx, nil)
	if err != nil || !exists {
		release()
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, ErrInvalidToken
	}
	return h.ended, release, nil
}

// CheckHolds ends every held session that storage no longer has. The change feed
// normally does this as a session is deleted, but it can drop notifications under load.
func (s *Service) CheckHolds(ctx context.Context) error {
	held := s.held()
	if len(held) == 0 {
		return nil
	}
	var sessions []Session
	if err := s.NewRetrieve().
		Where(MatchKeys(held...)).
		Entries(&sessions).
		Exec(ctx, nil); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	stored := make(set.Set[SessionKey], len(sessions))
	for _, e := range sessions {
		stored.Add(e.Key)
	}
	for _, key := range held {
		if !stored.Contains(key) {
			s.endHold(key)
		}
	}
	return nil
}

func (s *Service) endHold(key SessionKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h, ok := s.mu.holds[key]; ok && !h.over {
		h.over = true
		close(h.ended)
	}
}

func (s *Service) endDeletedHolds(
	_ context.Context,
	changes iter.Seq[change.Change[SessionKey, Session]],
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range changes {
		if ch.Variant != change.VariantDelete {
			continue
		}
		if h, ok := s.mu.holds[ch.Key]; ok && !h.over {
			h.over = true
			close(h.ended)
		}
	}
}

// held skips ended holds.
func (s *Service) held() []SessionKey {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]SessionKey, 0, len(s.mu.holds))
	for k, h := range s.mu.holds {
		if !h.over {
			keys = append(keys, k)
		}
	}
	return keys
}

// NewRetrieve opens a query for sessions.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), baseTX: s.cfg.DB, indexes: s.indexes}
}

// NewWriter opens a [Writer] that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: gorp.OverrideTx(s.cfg.DB, tx)}
}

// Login checks the proof with the authenticator for its method and opens a session for
// the identity it proves. It returns the session and its token, which is shown once. A
// proof for a method with no authenticator is a validation error.
func (s *Service) Login(
	ctx context.Context,
	proof Proof,
) (Session, string, error) {
	method, err := proof.method()
	if err != nil {
		return Session{}, "", err
	}
	authenticator, ok := s.cfg.Authenticators[method]
	if !ok {
		return Session{}, "", errors.Wrapf(
			validate.ErrValidation,
			"method %s is not accepted",
			method,
		)
	}
	identity, err := authenticator.Authenticate(ctx, proof)
	if err != nil {
		return Session{}, "", err
	}
	session := Session{
		Key:        uuid.New(),
		Subject:    identity.Subject,
		Credential: identity.Credential,
		LastActive: s.now(),
	}
	token, hash, err := NewToken(tokenPrefix, session.Key)
	if err != nil {
		return Session{}, "", err
	}
	session.Hash = hash
	if err = s.table.NewCreate().Entry(&session).Exec(ctx, s.cfg.DB); err != nil {
		return Session{}, "", err
	}
	return session, token, nil
}

// Authenticate returns the session a token belongs to and counts the request as
// activity. It returns [ErrInvalidToken] when no session matches the token and
// [ErrExpiredToken] when the session has been idle for the idle timeout, in which case
// the session is deleted.
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	key, hash, err := ParseToken(tokenPrefix, token)
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&session).
		Exec(ctx, nil); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return Session{}, ErrInvalidToken
		}
		return Session{}, err
	}
	if subtle.ConstantTimeCompare(session.Hash, hash) != 1 {
		return Session{}, ErrInvalidToken
	}
	now := s.now()
	if !s.see(key, session.LastActive, now) {
		if err = s.Logout(ctx, key); err != nil {
			return Session{}, err
		}
		return Session{}, ErrExpiredToken
	}
	if time.Duration(now-session.LastActive) > s.cfg.IdleTimeout/4 {
		if err = s.touch(ctx, map[SessionKey]telem.TimeStamp{key: now}); err != nil {
			return Session{}, err
		}
		session.LastActive = now
	}
	return session, nil
}

func (s *Service) lastSeen(key SessionKey) telem.TimeStamp {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mu.seen[key]
}

// see marks key as active at now and returns true, or returns false without a mark
// when the session has been idle for the idle timeout since stored or its last mark.
func (s *Service) see(key SessionKey, stored, now telem.TimeStamp) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Duration(now-max(stored, s.mu.seen[key])) > s.cfg.IdleTimeout {
		return false
	}
	s.mu.seen[key] = now
	return true
}

// forget also drops every session not in a non-nil keep.
func (s *Service) forget(keep set.Set[SessionKey], keys ...SessionKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		delete(s.mu.seen, k)
	}
	if keep == nil {
		return
	}
	for k := range s.mu.seen {
		if !keep.Contains(k) {
			delete(s.mu.seen, k)
		}
	}
}

// Logout ends the sessions with the given keys. Unknown keys are ignored.
func (s *Service) Logout(ctx context.Context, keys ...SessionKey) error {
	s.forget(nil, keys...)
	return s.table.NewDelete().
		Where(gorp.MatchKeys[SessionKey, Session](keys...)).
		Exec(ctx, s.cfg.DB)
}

// Sweep writes the activity of every held session, then ends every other session whose
// stored activity is older than the idle timeout plus a quarter. A node writes a
// session's activity up to a quarter period late, so a younger stored mark may hide a
// session that is live on another node.
func (s *Service) Sweep(ctx context.Context) error {
	now := s.now()
	cutoff := now - telem.TimeStamp(s.cfg.IdleTimeout+s.cfg.IdleTimeout/4)
	// collected before the read, so a hold registered after it is never ended here
	held := set.New(s.held()...)
	var all []Session
	if err := s.NewRetrieve().Entries(&all).Exec(ctx, nil); err != nil {
		return err
	}
	keep := make(set.Set[SessionKey], len(all))
	marks := make(map[SessionKey]telem.TimeStamp)
	var idle []SessionKey
	for _, e := range all {
		keep.Add(e.Key)
		if held.Contains(e.Key) {
			marks[e.Key] = now
			continue
		}
		if e.LastActive >= cutoff {
			continue
		}
		if seen := s.lastSeen(e.Key); seen >= cutoff {
			marks[e.Key] = seen
			continue
		}
		idle = append(idle, e.Key)
	}
	for key := range held {
		if !keep.Contains(key) {
			s.endHold(key)
		}
	}
	if err := s.touch(ctx, marks); err != nil {
		return err
	}
	s.forget(keep, idle...)
	if len(idle) == 0 {
		return nil
	}
	s.cfg.L.Info("ending idle sessions", zap.Int("count", len(idle)))
	return s.Logout(ctx, idle...)
}

func (s *Service) now() telem.TimeStamp { return telem.NewTimeStamp(s.cfg.Clock.Now()) }

func (s *Service) touch(
	ctx context.Context,
	marks map[SessionKey]telem.TimeStamp,
) error {
	if len(marks) == 0 {
		return nil
	}
	keys := slices.Collect(maps.Keys(marks))
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var sessions []Session
		// a session that ended since the caller read it has no activity to mark
		if err := s.table.NewRetrieve().
			Where(gorp.MatchKeys[SessionKey, Session](keys...)).
			Entries(&sessions).
			Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		for i := range sessions {
			sessions[i].LastActive = marks[sessions[i].Key]
		}
		return s.table.NewCreate().Entries(&sessions).Exec(ctx, tx)
	})
}

// Writer removes credentials and the sessions they opened in one transaction. It makes
// no identity check: the caller authorizes the change.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// DeleteFor removes every credential of the given subjects through each authenticator,
// then ends their sessions.
func (w Writer) DeleteFor(ctx context.Context, subjects ...ontology.ID) error {
	for _, a := range w.svc.cfg.Authenticators {
		if err := a.DeleteFor(ctx, w.tx, subjects...); err != nil {
			return err
		}
	}
	return w.svc.table.NewDelete().
		Where(MatchSubjects(subjects...)(w.svc.NewRetrieve())).
		Exec(ctx, w.tx)
}

// RevokeCredential ends every session the credentials opened, except those in keep.
func (w Writer) RevokeCredential(
	ctx context.Context,
	keep []SessionKey,
	credentials ...uuid.UUID,
) error {
	kept := set.New(keep...)
	return w.svc.table.NewDelete().
		Where(And(
			MatchCredentials(credentials...),
			Match(func(_ gorp.Context, _ Retrieve, e *Session) (bool, error) {
				return !kept.Contains(e.Key), nil
			}),
		)(w.svc.NewRetrieve())).
		Exec(ctx, w.tx)
}
