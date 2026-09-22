// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/base64"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// State is the verdict the service reached on the grants it holds.
type State string

const (
	// StateOK means a grant covers this Core.
	StateOK State = "ok"
	// StateMissing means no grant applies to this Core.
	StateMissing State = "missing"
	// StateExpired means the grant that applies no longer covers this Core.
	StateExpired State = "expired"
)

// Info is what the service knows about this Core's grant.
type Info struct {
	// State is the verdict.
	State State `json:"state" msgpack:"state"`
	// Warning is set while the state is ok but a change is near or past.
	Warning string `json:"warning,omitempty" msgpack:"warning,omitempty"`
	// Host identifies this machine.
	Host Host `json:"host" msgpack:"host"`
	// Grant is the grant that applies, if any.
	Grant *Grant `json:"grant,omitempty" msgpack:"grant,omitempty"`
}

// ServiceConfig is the configuration for a verification service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
	// DB is the database that stores accepted tokens and the clock mark.
	//
	// [REQUIRED]
	kv.DB
	// Verifier is a token to accept when the service opens.
	//
	// [OPTIONAL] - Defaults to ""
	Verifier string
	// Anchors is the key set tokens are verified against.
	//
	// [OPTIONAL] - Defaults to the production keys.
	Anchors Anchors
	// Version is this Core's version, checked against a grant's version ceiling. An
	// empty version passes every ceiling.
	//
	// [OPTIONAL] - Defaults to ""
	Version string
	// Now returns the current time.
	//
	// [OPTIONAL] - Defaults to time.Now
	Now func() time.Time
	// CheckInterval is how often the service records the clock and repeats its
	// warning.
	//
	// [OPTIONAL] - Defaults to 1 hour
	CheckInterval time.Duration
	// WarningTime is how long before a grant stops applying the warning starts.
	//
	// [OPTIONAL] - Defaults to 1 week
	WarningTime time.Duration
	// Grace is how long after a grant stops applying it still counts.
	//
	// [OPTIONAL] - Defaults to 14 days
	Grace time.Duration
	// Rollback is how far behind the recorded clock the current clock may fall before
	// every grant is treated as expired.
	//
	// [OPTIONAL] - Defaults to 24 hours
	Rollback time.Duration
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate validates the configuration for use in the service.
func (c ServiceConfig) Validate() error {
	v := validate.New("channel.verification")
	v.NotNil("db", c.DB)
	v.NotNil("anchors", c.Anchors)
	v.NotNil("now", c.Now)
	v.NonZero("check_interval", c.CheckInterval)
	v.NonZero("warning_time", c.WarningTime)
	v.NonZero("grace", c.Grace)
	v.NonZero("rollback", c.Rollback)
	return v.Error()
}

// Override replaces fields on c with valid fields from other.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Verifier = override.String(c.Verifier, other.Verifier)
	c.Anchors = override.Nil(c.Anchors, other.Anchors)
	c.Version = override.String(c.Version, other.Version)
	c.Now = override.Nil(c.Now, other.Now)
	c.CheckInterval = override.Numeric(c.CheckInterval, other.CheckInterval)
	c.WarningTime = override.Numeric(c.WarningTime, other.WarningTime)
	c.Grace = override.Numeric(c.Grace, other.Grace)
	c.Rollback = override.Numeric(c.Rollback, other.Rollback)
	return c
}

// DefaultServiceConfig is the default configuration for the verification service.
var DefaultServiceConfig = ServiceConfig{
	Anchors:       anchors,
	Now:           time.Now,
	CheckInterval: time.Hour,
	WarningTime:   7 * 24 * time.Hour,
	Grace:         14 * 24 * time.Hour,
	Rollback:      24 * time.Hour,
}

var (
	// prefix keys the accepted tokens. The stored value is the token itself.
	prefix = []byte("bGljZW5zZUtleQ==/")
	// legacyKey held the previous format; it is removed on open.
	legacyKey = []byte("bGljZW5zZUtleQ==")
	// markKey holds the latest clock reading the service has recorded.
	markKey = []byte("aGlnaFdhdGVy")
)

// Service verifies the grant a Core runs under and gates the API on it.
type Service struct {
	cfg      ServiceConfig
	host     Host
	shutdown io.Closer
	// rolledBack is set when the clock at open fell behind the recorded mark. The
	// service never records the clock again in that state.
	rolledBack bool
	mu         struct {
		sync.RWMutex
		info Info
	}
}

var _ io.Closer = &Service{}

// OpenService opens the service: it reads the host, checks the clock against the
// recorded mark, loads the token that fits this host, and accepts cfg.Verifier when
// set. A verifier that fails to verify is an error; a Core with no grant opens in
// StateMissing.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.host, err = readHost(); err != nil {
		cfg.L.Warn("failed to read network interfaces", zap.Error(err))
		s.host = Host{}
	}
	s.mu.info = Info{State: StateMissing, Host: s.host}
	if err = s.checkClock(ctx); err != nil {
		return nil, err
	}
	if err = cfg.Delete(ctx, legacyKey); err != nil {
		return nil, err
	}
	if err = s.load(ctx); err != nil {
		return nil, err
	}
	if cfg.Verifier != "" {
		if _, err = s.Activate(ctx, cfg.Verifier); err != nil {
			return nil, err
		}
	}
	s.logState()
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	s.shutdown = signal.NewHardShutdown(sCtx, cancel)
	sCtx.Go(
		s.monitor,
		signal.WithRetryOnPanic(),
		signal.WithBaseRetryInterval(2*time.Second),
		signal.WithRetryScale(1.1),
		signal.WithKey("verification"),
	)
	return s, nil
}

// Close should be called when the service is no longer needed.
func (s *Service) Close() error { return s.shutdown.Close() }

// Retrieve returns what the service knows about this Core's grant.
func (s *Service) Retrieve() Info {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.info
}

// Check returns nil while a grant covers this Core, ErrMissing while none applies,
// and ErrExpired while the one that applies no longer covers it.
func (s *Service) Check() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.info.err()
}

func (i Info) err() error {
	switch i.State {
	case StateMissing:
		return ErrMissing
	case StateExpired:
		if i.Warning != "" {
			return errors.Wrap(ErrExpired, i.Warning)
		}
		return ErrExpired
	}
	return nil
}

// Activate verifies token, checks that it fits this host and still applies, stores
// it, and moves the service to StateOK. The stored token loads on the next open.
// Returns ErrInvalid, ErrHost, or ErrExpired when the token is refused.
func (s *Service) Activate(ctx context.Context, token string) (Info, error) {
	grant, err := Verify(s.cfg.Anchors, token)
	if err != nil {
		return Info{}, err
	}
	if grant.Exp == nil && grant.Mv == nil {
		return Info{}, errors.Wrap(ErrInvalid, base64.MustDecode(
			"YSBsaWNlbnNlIHdpdGhvdXQgYW4gZXhwaXJ5IG11c3QgY2FycnkgYSBtYXhpbXVtIHZlcnNpb24=",
		))
	}
	if grant.Mv != nil {
		if _, _, ok := parseMinor(*grant.Mv); !ok {
			return Info{}, errors.Wrapf(ErrInvalid, "bad version ceiling %q", *grant.Mv)
		}
	}
	if !s.host.Covers(grant.Fs, grant.Fp) {
		return Info{}, ErrHost
	}
	info := s.evaluate(grant)
	if info.State != StateOK {
		return Info{}, info.err()
	}
	key := append(append([]byte{}, prefix...), grant.Jti.String()...)
	if err = s.cfg.Set(ctx, key, []byte(token)); err != nil {
		return Info{}, err
	}
	s.mu.Lock()
	s.mu.info = info
	s.mu.Unlock()
	s.logState()
	return info, nil
}

// CheckOverflow returns ErrTooMany when inUse external channels exceed the grant's
// cap. A Core without a covering grant is idle behind the API gate, so the cap does
// not apply to it.
func (s *Service) CheckOverflow(inUse types.Uint20) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info := s.mu.info
	if info.State != StateOK || info.Grant == nil || info.Grant.Ch == 0 {
		return nil
	}
	if uint32(inUse) > info.Grant.Ch {
		return newTooManyError(info.Grant.Ch)
	}
	return nil
}

// checkClock compares the clock with the recorded mark and moves the mark forward.
// A clock more than Rollback behind the mark marks the service rolled back.
func (s *Service) checkClock(ctx context.Context) error {
	now := s.cfg.Now()
	raw, closer, err := s.cfg.Get(ctx, markKey)
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	if err == nil {
		// A mark of another width is treated as absent and overwritten.
		var mark time.Time
		if len(raw) == 8 {
			mark = time.Unix(0, int64(binary.LittleEndian.Uint64(raw)))
		}
		if err = closer.Close(); err != nil {
			return err
		}
		if mark.Sub(now) > s.cfg.Rollback {
			s.rolledBack = true
			return nil
		}
		if !now.After(mark) {
			return nil
		}
	}
	return s.recordClock(ctx, now)
}

func (s *Service) recordClock(ctx context.Context, now time.Time) error {
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, uint64(now.UnixNano()))
	return s.cfg.Set(ctx, markKey, raw)
}

// load picks the stored token that fits this host. A token that still applies wins
// over one that no longer does, so an expired grant is reported only when no other
// covers the Core.
func (s *Service) load(ctx context.Context) error {
	iter, err := s.cfg.OpenIterator(kv.IterPrefix(prefix))
	if err != nil {
		return err
	}
	var chosen *Info
	for iter.First(); iter.Valid(); iter.Next() {
		grant, err := Verify(s.cfg.Anchors, string(iter.Value()))
		if err != nil {
			s.cfg.L.Warn(
				"skipping a stored token that no longer verifies",
				zap.Error(err),
			)
			continue
		}
		if !s.host.Covers(grant.Fs, grant.Fp) {
			continue
		}
		info := s.evaluate(grant)
		if chosen == nil || (chosen.State != StateOK && info.State == StateOK) {
			chosen = &info
		}
	}
	if err = iter.Close(); err != nil {
		return err
	}
	if chosen != nil {
		s.mu.info = *chosen
	}
	return nil
}

var (
	warnExpiresTemplate = base64.MustDecode("bGljZW5zZSBleHBpcmVzIGluICVz")
	warnGraceTemplate   = base64.MustDecode(
		"bGljZW5zZSBleHBpcmVkIG9uICVzLCBncmFjZSBwZXJpb2QgZW5kcyBvbiAlcw==",
	)
	warnFallbackTemplate = base64.MustDecode(
		"c3Vic2NyaXB0aW9uIGVuZGVkIG9uICVzLCB0aGlzIHZlcnNpb24gaXMgY292ZXJlZCB1cCB0byAlcw==",
	)
	expiredVersionTemplate = base64.MustDecode(
		"bGljZW5zZSBjb3ZlcnMgdmVyc2lvbnMgdXAgdG8gJXMsIHRoaXMgQ29yZSBpcyAlcw==",
	)
	expiredClockTemplate = base64.MustDecode(
		"c3lzdGVtIGNsb2NrIGlzIGJlaGluZCB0aGUgbGFzdCByZWNvcmRlZCB0aW1lIGJ5IG1vcmUgdGhhbiAlcywgdHJlYXRpbmcgdGhlIGxpY2Vuc2UgYXMgZXhwaXJlZA==",
	)
)

// evaluate decides the state a grant puts this Core in at the current time.
func (s *Service) evaluate(grant Grant) Info {
	info := Info{State: StateOK, Host: s.host, Grant: &grant}
	if s.rolledBack {
		info.State = StateExpired
		info.Warning = fmt.Sprintf(expiredClockTemplate, s.cfg.Rollback)
		return info
	}
	covered := grant.Mv == nil || versionCovered(s.cfg.Version, *grant.Mv)
	if grant.Exp == nil {
		if !covered {
			info.State = StateExpired
			info.Warning = fmt.Sprintf(expiredVersionTemplate, *grant.Mv, s.cfg.Version)
		}
		return info
	}
	now := s.cfg.Now()
	exp := time.Unix(int64(*grant.Exp), 0)
	if now.Before(exp) {
		if left := exp.Sub(now); left <= s.cfg.WarningTime {
			info.Warning = fmt.Sprintf(warnExpiresTemplate, left.Round(time.Minute))
		}
		return info
	}
	if graceEnd := exp.Add(s.cfg.Grace); now.Before(graceEnd) {
		info.Warning = fmt.Sprintf(
			warnGraceTemplate,
			exp.Format(time.DateOnly),
			graceEnd.Format(time.DateOnly),
		)
		return info
	}
	if grant.Mv != nil && covered {
		info.Warning = fmt.Sprintf(
			warnFallbackTemplate,
			exp.Format(time.DateOnly),
			*grant.Mv,
		)
		return info
	}
	info.State = StateExpired
	if grant.Mv != nil {
		info.Warning = fmt.Sprintf(expiredVersionTemplate, *grant.Mv, s.cfg.Version)
	}
	return info
}

// versionCovered reports whether version's major and minor are at most ceiling's. An
// unparseable version passes, so a development build is never gated by its own
// version string.
func versionCovered(version, ceiling string) bool {
	maj, min, ok := parseMinor(version)
	if !ok {
		return true
	}
	cMaj, cMin, ok := parseMinor(ceiling)
	if !ok {
		return false
	}
	return maj < cMaj || (maj == cMaj && min <= cMin)
}

// parseMinor reads the leading "major.minor" of a version string.
func parseMinor(version string) (major, minor int, ok bool) {
	parts := strings.SplitN(strings.TrimPrefix(version, "v"), ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	var err error
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, false
	}
	if minor, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

var (
	logActive      = base64.MustDecode("bGljZW5zZSBhY3RpdmU=")
	logCapTemplate = base64.MustDecode(
		"bGljZW5zZSBhY3RpdmUsIGxpbWl0IGlzICVkIGNoYW5uZWxz",
	)
)

func (s *Service) logState() {
	info := s.Retrieve()
	switch info.State {
	case StateOK:
		if info.Grant.Ch == 0 {
			s.cfg.L.Info(logActive)
		} else {
			s.cfg.L.Infof(logCapTemplate, info.Grant.Ch)
		}
		if info.Warning != "" {
			s.cfg.L.Warn(info.Warning)
		}
	case StateMissing:
		s.cfg.L.Warn(ErrMissing.Error())
	case StateExpired:
		s.cfg.L.Error(info.err().Error())
	}
}

// monitor records the clock and repeats the warning on every check interval. The
// state itself never changes here: a Core that opened covered stays covered until it
// restarts.
func (s *Service) monitor(ctx context.Context) error {
	ticker := time.NewTicker(s.cfg.CheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !s.rolledBack {
				if err := s.recordClock(ctx, s.cfg.Now()); err != nil {
					s.cfg.L.Warn("failed to record clock", zap.Error(err))
				}
			}
			if info := s.Retrieve(); info.State == StateOK && info.Warning != "" {
				s.cfg.L.Warn(info.Warning)
			}
		}
	}
}
