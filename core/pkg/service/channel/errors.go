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
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// NameErrors returns err with its message naming the channels it mentions by their
// records. A channel error reads `channel "<name>"`, or `deleted channel <key>`
// when no record has the key, and any other `<key>` that Cesium prints reads
// `"<name>"`. A channel for which visible returns false keeps its key, and a nil
// visible names every channel. The returned error matches err under errors.Is and
// errors.As.
func (s *Service) NameErrors(
	ctx context.Context,
	err error,
	visible func(Key) bool,
) error {
	if err == nil {
		return nil
	}
	errKeys := channelErrorKeys(err)
	keys := mentionedKeys(err.Error(), errKeys)
	if len(keys) == 0 {
		return err
	}
	var found []Channel
	if rErr := s.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		SkipMissingKeys().
		Entries(&found).
		Exec(ctx, s.db); rErr != nil {
		s.cfg.L.Warn("failed to name the channels of an error", zap.Error(rErr))
		return err
	}
	names := make(map[Key]string, len(found))
	for _, c := range found {
		names[c.Key()] = c.Name
	}
	labels := make(map[Key]string, len(keys))
	for _, k := range keys {
		if visible != nil && !visible(k) {
			continue
		}
		if name, ok := names[k]; ok {
			labels[k] = fmt.Sprintf("%q", name)
		} else if errKeys.Contains(k) {
			labels[k] = fmt.Sprintf("<%d>", k)
		}
	}
	if len(labels) == 0 {
		return err
	}
	return rename(err, func(msg string) string { return relabel(msg, labels, names) })
}

// relabel replaces each key in msg that labels holds, with any name the Core printed in
// brackets before it, by its label. A channel error of a key with no record reads
// `deleted channel <key>`.
func relabel(msg string, labels, names map[Key]string) string {
	return keyToken.ReplaceAllStringFunc(msg, func(match string) string {
		m := keyToken.FindStringSubmatch(match)
		k, pErr := strconv.ParseUint(m[2], 10, 32)
		if pErr != nil {
			return match
		}
		label, ok := labels[Key(k)]
		if !ok {
			return match
		}
		if _, named := names[Key(k)]; !named {
			if m[1] == "" {
				return match
			}
			return "deleted " + m[1] + label
		}
		return m[1] + label
	})
}

// keyToken matches a channel key as Cesium prints it, `channel <key>` or `<key>`, with
// any name that the Core prints in brackets before it. Group 1 holds the prefix.
var keyToken = regexp.MustCompile(`(channel )?(?:\[[^\]]*\])?<(\d+)>`)

// mentionedKeys returns the keys of errKeys and of the key tokens in msg, once each.
func mentionedKeys(msg string, errKeys set.Set[Key]) []Key {
	seen := set.New[Key]()
	keys := make([]Key, 0, len(errKeys))
	add := func(k Key) {
		if !seen.Contains(k) {
			seen.Add(k)
			keys = append(keys, k)
		}
	}
	for k := range errKeys {
		add(k)
	}
	for _, m := range keyToken.FindAllStringSubmatch(msg, -1) {
		if k, pErr := strconv.ParseUint(m[2], 10, 32); pErr == nil {
			add(Key(k))
		}
	}
	return keys
}

// channelErrorKeys returns the key of each channel error in err, from Cesium or from
// the distribution layer.
func channelErrorKeys(err error) set.Set[Key] {
	keys := set.New[Key]()
	walk(err, func(e error) {
		var (
			ce cesium.ChannelError
			de distchannel.Error
		)
		if errors.As(e, &ce) {
			keys.Add(Key(ce.Key))
		}
		if errors.As(e, &de) {
			keys.Add(de.Key)
		}
	})
	return keys
}

// walk calls fn on err and on every error that err wraps.
func walk(err error, fn func(error)) {
	if err == nil {
		return
	}
	fn(err)
	if members := errors.UnwrapMulti(err); members != nil && isJoin(err) {
		for _, m := range members {
			walk(m, fn)
		}
		return
	}
	walk(errors.UnwrapOnce(err), fn)
}

func isJoin(err error) bool {
	_, ok := err.(interface{ Unwrap() []error })
	return ok
}

// rename returns err with r applied to its message, down to the last error in the chain
// whose message r changes. Below that error it hands back the original chain, so the
// sentinels of err stay identical for errors.Is and for the error codec.
func rename(err error, r func(string) string) error {
	if err == nil || r(err.Error()) == err.Error() {
		return err
	}
	if isJoin(err) {
		return namedJoin{named{err: err, r: r}}
	}
	return named{err: err, r: r}
}

// named is an error whose message names the channels of the error it renders.
type named struct {
	err error
	r   func(string) string
}

// Error implements error.
func (n named) Error() string { return n.r(n.err.Error()) }

// Unwrap returns the cause of the rendered error, named in turn.
func (n named) Unwrap() error { return rename(errors.UnwrapOnce(n.err), n.r) }

// Is reports whether the rendered error is target.
func (n named) Is(target error) bool {
	t := reflect.TypeOf(target)
	return t != nil && t.Comparable() && t == reflect.TypeOf(n.err) && n.err == target
}

// As sets target to the rendered error when it can hold it.
func (n named) As(target any) bool {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return false
	}
	if !reflect.TypeOf(n.err).AssignableTo(v.Elem().Type()) {
		return false
	}
	v.Elem().Set(reflect.ValueOf(n.err))
	return true
}

// namedJoin is a named join of errors.
type namedJoin struct{ named }

// Unwrap returns the members of the rendered join, each named in turn.
func (n namedJoin) Unwrap() []error {
	members := errors.UnwrapMulti(n.err)
	out := make([]error, len(members))
	for i, m := range members {
		out[i] = rename(m, n.r)
	}
	return out
}
