// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for a license service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
	// DB is the database that stores accepted license keys and the clock mark.
	//
	// [REQUIRED]
	kv.DB
	// Key is a license key to accept when the service opens.
	//
	// [OPTIONAL] - Defaults to ""
	Key string
	// Anchors is the key set license keys are verified against.
	//
	// [OPTIONAL] - Defaults to the production keys.
	Anchors Anchors
	// Version is this Core's version, checked against a license's version ceiling. An
	// empty version passes every ceiling.
	//
	// [OPTIONAL] - Defaults to ""
	Version string
	// Now returns the current time.
	//
	// [OPTIONAL] - Defaults to time.Now
	Now func() time.Time
	// CheckInterval is how often the service records the clock and repeats its warning.
	// While the clock is behind, the service checks it every minute instead.
	//
	// [OPTIONAL] - Defaults to 1 hour
	CheckInterval time.Duration
	// WarningTime is how long before a license stops applying the warning starts.
	//
	// [OPTIONAL] - Defaults to 1 week
	WarningTime time.Duration
	// Grace is how long after a license stops applying it still counts.
	//
	// [OPTIONAL] - Defaults to 14 days
	Grace time.Duration
	// Rollback is how far behind the recorded clock the current clock may fall before
	// licenses with an expiry are treated as expired.
	//
	// [OPTIONAL] - Defaults to 24 hours
	Rollback time.Duration
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate validates the configuration for use in the service.
func (c ServiceConfig) Validate() error {
	v := validate.New("license")
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
	c.Key = override.String(c.Key, other.Key)
	c.Anchors = override.Nil(c.Anchors, other.Anchors)
	c.Version = override.String(c.Version, other.Version)
	c.Now = override.Nil(c.Now, other.Now)
	c.CheckInterval = override.Numeric(c.CheckInterval, other.CheckInterval)
	c.WarningTime = override.Numeric(c.WarningTime, other.WarningTime)
	c.Grace = override.Numeric(c.Grace, other.Grace)
	c.Rollback = override.Numeric(c.Rollback, other.Rollback)
	return c
}

// DefaultServiceConfig is the default configuration for the license service.
var DefaultServiceConfig = ServiceConfig{
	Anchors:       anchors,
	Now:           time.Now,
	CheckInterval: time.Hour,
	WarningTime:   7 * 24 * time.Hour,
	Grace:         14 * 24 * time.Hour,
	Rollback:      24 * time.Hour,
}

var (
	// prefix keys the accepted license keys. The stored value is the license key.
	prefix = []byte("license/")
	// legacyKey is where Cores before signed licenses stored a key. Open deletes it.
	legacyKey = []byte("bGljZW5zZUtleQ==")
	// markKey holds the latest clock reading the service has recorded.
	markKey = []byte("highWater")
)

// Service verifies the license a Core runs under.
type Service struct {
	cfg         ServiceConfig
	fingerprint Fingerprint
	shutdown    io.Closer
	// clockBehind is set while the clock is more than Rollback behind the recorded
	// mark.
	clockBehind atomic.Bool
	// loadMu serializes load, so the last load to publish has seen every stored key.
	loadMu sync.Mutex
	mu     struct {
		sync.RWMutex
		info Info
	}
}

var _ io.Closer = &Service{}

// OpenService opens the service and activates cfg.Key when set. It returns an error
// when cfg.Key is refused or the machine fingerprint cannot be read. A Core with no
// license opens in StateMissing.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.fingerprint, err = readFingerprint(); err != nil {
		return nil, errors.Wrap(err, "failed to read the machine fingerprint")
	}
	s.mu.info = Info{State: StateMissing, Fingerprint: s.fingerprint}
	if err = s.syncClock(ctx); err != nil {
		return nil, err
	}
	if err = cfg.Delete(ctx, legacyKey); err != nil {
		return nil, err
	}
	if err = s.load(ctx); err != nil {
		return nil, err
	}
	if cfg.Key == "" {
		s.logState()
	} else if _, err = s.Activate(ctx, cfg.Key); err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	s.shutdown = signal.NewHardShutdown(sCtx, cancel)
	sCtx.Go(
		s.monitor,
		signal.WithRetryOnPanic(),
		signal.WithBaseRetryInterval(2*time.Second),
		signal.WithRetryScale(1.1),
		signal.WithKey("license"),
	)
	return s, nil
}

// Close should be called when the service is no longer needed.
func (s *Service) Close() error { return s.shutdown.Close() }

// Retrieve returns what the service knows about this Core's license.
func (s *Service) Retrieve() Info {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.info
}

// Check returns nil while a license covers this Core, ErrMissing while none applies,
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

// Activate verifies key, checks that it fits this machine and still covers it, and
// stores it. The service then applies the stored key open would pick, which is key
// unless a newer one is stored, and returns its info. Returns ErrInvalid,
// ErrFingerprint, or ErrExpired when the key is refused.
func (s *Service) Activate(ctx context.Context, key string) (Info, error) {
	lic, err := Verify(s.cfg.Anchors, key)
	if err != nil {
		return Info{}, err
	}
	if !s.fingerprint.Covers(lic.FingerprintScheme, lic.Fingerprints) {
		return Info{}, ErrFingerprint
	}
	info := s.evaluate(lic)
	if info.State != StateOk {
		return Info{}, info.err()
	}
	entry := append(append([]byte{}, prefix...), lic.Jti.String()...)
	if err = s.cfg.Set(ctx, entry, []byte(key)); err != nil {
		return Info{}, err
	}
	if err = s.load(ctx); err != nil {
		return Info{}, err
	}
	s.logState()
	return s.Retrieve(), nil
}

// CheckChannelLimit returns ErrTooMany when inUse external channels exceed the
// license's cap, and nil while no license covers the Core.
func (s *Service) CheckChannelLimit(inUse types.Uint20) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info := s.mu.info
	if info.State != StateOk || info.License.Channels == 0 {
		return nil
	}
	if uint32(inUse) > info.License.Channels {
		return newTooManyError(info.License.Channels)
	}
	return nil
}

// syncClock compares the clock with the recorded mark and moves the mark forward, never
// back. A clock more than Rollback behind the mark sets clockBehind until it catches
// up.
func (s *Service) syncClock(ctx context.Context) error {
	now := s.cfg.Now()
	raw, closer, err := s.cfg.Get(ctx, markKey)
	if errors.Is(err, query.ErrNotFound) {
		return s.recordClock(ctx, now)
	}
	if err != nil {
		return err
	}
	if len(raw) != 8 {
		return errors.Join(
			errors.Newf("license clock mark has %d bytes, want 8", len(raw)),
			closer.Close(),
		)
	}
	mark := time.Unix(0, int64(binary.LittleEndian.Uint64(raw)))
	if err = closer.Close(); err != nil {
		return err
	}
	behind := mark.Sub(now) > s.cfg.Rollback
	if s.clockBehind.Swap(behind) != behind {
		fields := []zap.Field{zap.Time("now", now), zap.Time("recorded", mark)}
		if behind {
			s.cfg.L.Warn(fmt.Sprintf(
				"system clock is more than %s behind the last recorded time, so "+
					"license terms cannot be checked until it catches up",
				s.cfg.Rollback,
			), fields...)
		} else {
			s.cfg.L.Info(
				"system clock caught up with the last recorded time, reloading the "+
					"license",
				fields...,
			)
		}
	}
	if now.After(mark) {
		return s.recordClock(ctx, now)
	}
	return nil
}

func (s *Service) recordClock(ctx context.Context, now time.Time) error {
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint64(raw, uint64(now.UnixNano()))
	return s.cfg.Set(ctx, markKey, raw)
}

// load applies the stored key that fits this machine. A key that still applies wins
// over one that no longer does, then the most recently issued wins.
func (s *Service) load(ctx context.Context) error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()
	iter, err := s.cfg.OpenIterator(kv.IterPrefix(prefix))
	if err != nil {
		return err
	}
	var chosen *Info
	for iter.First(); iter.Valid(); iter.Next() {
		lic, err := Verify(s.cfg.Anchors, string(iter.Value()))
		if err != nil {
			s.cfg.L.Warn(
				"skipping a stored license key that no longer verifies",
				zap.Error(err),
			)
			continue
		}
		if !s.fingerprint.Covers(lic.FingerprintScheme, lic.Fingerprints) {
			s.cfg.L.Warn(
				"skipping a stored license key issued for another machine",
				zap.Stringer("jti", lic.Jti),
			)
			continue
		}
		info := s.evaluate(lic)
		if chosen == nil || better(info, *chosen) {
			chosen = &info
		}
	}
	if err = iter.Close(); err != nil {
		return err
	}
	if chosen != nil {
		s.mu.Lock()
		s.mu.info = *chosen
		s.mu.Unlock()
	}
	return nil
}

func better(a, b Info) bool {
	if okA, okB := a.State == StateOk, b.State == StateOk; okA != okB {
		return okA
	}
	return a.License.Iat > b.License.Iat
}

const clockTemplate = "system clock is more than %s behind the last recorded time, " +
	"so the license term cannot be checked"

// evaluate decides the state a license puts this Core in at the current time.
func (s *Service) evaluate(lic License) Info {
	info := Info{State: StateOk, Fingerprint: s.fingerprint, License: &lic}
	termUnknown := lic.Exp != nil && s.clockBehind.Load()
	if lic.Exp != nil && !termUnknown {
		now, exp := s.cfg.Now(), time.Unix(int64(*lic.Exp), 0)
		if now.Before(exp) {
			if exp.Sub(now) <= s.cfg.WarningTime {
				info.Warning = "license expires on " + exp.Format(time.DateOnly)
			}
			return info
		}
		if graceEnd := exp.Add(s.cfg.Grace); now.Before(graceEnd) {
			info.Warning = fmt.Sprintf(
				"license expired on %s, grace period ends on %s",
				exp.Format(time.DateOnly),
				graceEnd.Format(time.DateOnly),
			)
			return info
		}
	}
	if lic.MaxVersion != nil && versionCovered(s.cfg.Version, *lic.MaxVersion) {
		switch {
		case termUnknown:
			info.Warning = fmt.Sprintf(clockTemplate, s.cfg.Rollback)
		case lic.Exp != nil:
			info.Warning = fmt.Sprintf(
				"subscription ended on %s, this version is covered up to %s",
				time.Unix(int64(*lic.Exp), 0).Format(time.DateOnly),
				*lic.MaxVersion,
			)
		}
		return info
	}
	info.State = StateExpired
	switch {
	case termUnknown:
		info.Warning = fmt.Sprintf(clockTemplate, s.cfg.Rollback)
	case lic.MaxVersion != nil:
		info.Warning = fmt.Sprintf(
			"license covers versions up to %s, this Core is %s",
			*lic.MaxVersion,
			s.cfg.Version,
		)
	}
	return info
}

// versionCovered reports whether version's major and minor are at most ceiling's. An
// unparseable version, such as an empty one, passes every ceiling.
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

// clockRecheck is how often the service checks a clock that is behind.
const clockRecheck = time.Minute

func (s *Service) logState() {
	info := s.Retrieve()
	switch info.State {
	case StateOk:
		if info.License.Channels == 0 {
			s.cfg.L.Info("license active")
		} else {
			s.cfg.L.Infof("license active, limit is %d channels", info.License.Channels)
		}
		if info.Warning != "" {
			s.cfg.L.Warn(info.Warning)
		}
	case StateExpired:
		s.cfg.L.Error(info.err().Error())
	}
}

// monitor checks the clock and repeats the warning on every check interval. The state
// changes here only when a clock that was behind catches up: a Core that opened covered
// stays covered until it restarts.
func (s *Service) monitor(ctx context.Context) error {
	for {
		interval := s.cfg.CheckInterval
		if s.clockBehind.Load() {
			interval = min(interval, clockRecheck)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
		wasBehind := s.clockBehind.Load()
		if err := s.syncClock(ctx); err != nil {
			s.cfg.L.Warn("failed to check the clock", zap.Error(err))
			continue
		}
		if s.clockBehind.Load() {
			continue
		}
		if wasBehind {
			if err := s.load(ctx); err != nil {
				s.cfg.L.Warn("failed to reload the license", zap.Error(err))
				continue
			}
			s.logState()
			continue
		}
		if info := s.Retrieve(); info.State == StateOk && info.Warning != "" {
			s.cfg.L.Warn(info.Warning)
		}
	}
}
