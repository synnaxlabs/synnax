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
	"sync"
	"time"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
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
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("auth")
	v.NotNil("db", c.DB)
	v.Ternary("authenticators", len(c.Authenticators) == 0, "must not be empty")
	v.Positive("idle_timeout", c.IdleTimeout)
	v.NotNil("clock", c.Clock)
	return v.Error()
}

// Service logs subjects in and checks the session tokens their requests carry.
type Service struct {
	cfg      ServiceConfig
	table    *gorp.Table[SessionKey, Session]
	indexes  indexes
	shutdown io.Closer
	// mu guards seen.
	mu sync.Mutex
	// seen leads the stored activity by up to a quarter of the idle timeout.
	seen map[SessionKey]telem.TimeStamp
}

// OpenService opens a [Service] with the given configurations. It sweeps idle sessions
// in the background until Close.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(
		ServiceConfig{IdleTimeout: DefaultIdleTimeout, Clock: xtime.Real},
		cfgs...,
	)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:     cfg,
		indexes: newIndexes(),
		seen:    make(map[SessionKey]telem.TimeStamp),
	}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[SessionKey, Session]{
		DB:              cfg.DB,
		Indexes:         s.indexes.all(),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
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
	return s, nil
}

// Close stops the sweep and releases the service's resources.
func (s *Service) Close() error {
	return errors.Combine(s.shutdown.Close(), s.table.Close())
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
	method, err := proof.Method()
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
	token, hash, err := newToken(session.Key)
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
	key, hash, err := parseToken(token)
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
	last := max(session.LastActive, s.see(key, now))
	if time.Duration(now-last) > s.cfg.IdleTimeout {
		if err = s.Logout(ctx, key); err != nil {
			return Session{}, err
		}
		return Session{}, ErrExpiredToken
	}
	if time.Duration(now-session.LastActive) > s.cfg.IdleTimeout/4 {
		if err = s.touch(ctx, key, now); err != nil {
			return Session{}, err
		}
		session.LastActive = now
	}
	return session, nil
}

func (s *Service) lastSeen(key SessionKey) telem.TimeStamp {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seen[key]
}

// see marks key as active at and returns the previous mark.
func (s *Service) see(key SessionKey, at telem.TimeStamp) telem.TimeStamp {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.seen[key]
	s.seen[key] = at
	return prev
}

// forget also drops every session not in a non-nil keep.
func (s *Service) forget(keep set.Set[SessionKey], keys ...SessionKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		delete(s.seen, k)
	}
	if keep == nil {
		return
	}
	for k := range s.seen {
		if !keep.Contains(k) {
			delete(s.seen, k)
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

// Sweep ends every session whose stored activity is older than the idle timeout plus
// a quarter. A node writes a session's activity up to a quarter period late, so a
// younger stored mark may hide a session that is live on another node.
func (s *Service) Sweep(ctx context.Context) error {
	cutoff := s.now() - telem.TimeStamp(s.cfg.IdleTimeout+s.cfg.IdleTimeout/4)
	var all []Session
	if err := s.NewRetrieve().Entries(&all).Exec(ctx, nil); err != nil {
		return err
	}
	keep := make(set.Set[SessionKey], len(all))
	var idle []SessionKey
	for _, e := range all {
		keep.Add(e.Key)
		if e.LastActive >= cutoff {
			continue
		}
		if seen := s.lastSeen(e.Key); seen >= cutoff {
			if err := s.touch(ctx, e.Key, seen); err != nil {
				return err
			}
			continue
		}
		idle = append(idle, e.Key)
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
	key SessionKey,
	now telem.TimeStamp,
) error {
	return s.table.NewUpdate().
		Where(gorp.MatchKeys[SessionKey, Session](key)).
		Change(func(_ gorp.Context, e Session) Session {
			e.LastActive = now
			return e
		}).
		Exec(ctx, s.cfg.DB)
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
