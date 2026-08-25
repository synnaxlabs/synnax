// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
)

// state accumulates what the referential checks reference across tables. The first
// scanning pass fills it; the second reads it.
type state struct {
	// db is the store every scan reads through.
	db kv.DB
	// codec decodes stored values.
	codec encoding.Codec
	// violations aggregates repeats of one condition into a single finding.
	violations map[violationKey]*violation
	// order preserves the order conditions were first observed.
	order []violationKey
	// decode counts undecodable entries per table.
	decode map[string]*violation
	// reportDecode is true during the first pass, when decode failures are counted.
	reportDecode bool
	// channels holds every stored channel by key.
	channels map[channel.Key]channel.Channel
	// racks holds the key of every stored rack.
	racks set.Set[rack.Key]
	// ranges holds the key of every stored range.
	ranges set.Set[ranger.Key]
	// usernames holds the username of every stored user.
	usernames set.Set[string]
	// tasks holds every stored task key.
	tasks set.Set[task.Key]
	// resources holds the string form of every stored ontology resource ID.
	resources set.Set[string]
	// entities holds the keys of stored entities by the ontology type backing them.
	entities map[ontology.ResourceType]set.Set[string]
	// configs holds the keys of stored task config records, across every config
	// table.
	configs set.Set[uuid.UUID]
}

// newState builds an empty state reading through db with codec.
func newState(db kv.DB, codec encoding.Codec) *state {
	return &state{
		db:         db,
		codec:      codec,
		violations: make(map[violationKey]*violation),
		decode:     make(map[string]*violation),
		channels:   make(map[channel.Key]channel.Channel),
		racks:      make(set.Set[rack.Key]),
		ranges:     make(set.Set[ranger.Key]),
		usernames:  make(set.Set[string]),
		tasks:      make(set.Set[task.Key]),
		resources:  make(set.Set[string]),
		entities:   make(map[ontology.ResourceType]set.Set[string]),
		configs:    make(set.Set[uuid.UUID]),
	}
}

// violationKey identifies one aggregated condition.
type violationKey struct {
	// check is the check the condition belongs to.
	check Check
	// label describes the condition in one phrase.
	label string
}

// note records one failure of check under the given label. Repeats aggregate, so a
// store with many broken entries yields one finding per condition.
func (s *state) note(check Check, label, subject string) {
	key := violationKey{check: check, label: label}
	v, found := s.violations[key]
	if !found {
		v = &violation{}
		s.violations[key] = v
		s.order = append(s.order, key)
	}
	v.note(subject)
}

// noteN records count failures of check under the given label, for conditions counted
// outside the scanning passes.
func (s *state) noteN(check Check, label, subject string, count int) {
	s.note(check, label, subject)
	s.violations[violationKey{check: check, label: label}].count = count
}

// findings renders every aggregated condition as one finding, in the order the
// conditions were first observed.
func (s *state) findings() []Finding {
	findings := make([]Finding, 0, len(s.order))
	for _, key := range s.order {
		v := s.violations[key]
		findings = append(findings, newFinding(key.check, v.first, "%s", v.message(
			key.label,
		)))
	}
	return findings
}

// declare records that a table backs the given ontology type, so a resource of that
// type is judged against the table's keys rather than reported as unbacked. The zero
// type declares nothing.
func (s *state) declare(t ontology.ResourceType) {
	if t == "" {
		return
	}
	if _, found := s.entities[t]; !found {
		s.entities[t] = make(set.Set[string])
	}
}

// entity records that an entity backing the given resource ID is stored.
func (s *state) entity(id ontology.ID) {
	keys, found := s.entities[id.Type]
	if !found {
		keys = make(set.Set[string])
		s.entities[id.Type] = keys
	}
	keys.Add(id.Key)
}

// noteDecode counts one undecodable entry in the named table.
func (s *state) noteDecode(table, key string) {
	if !s.reportDecode {
		return
	}
	v, found := s.decode[table]
	if !found {
		v = &violation{}
		s.decode[table] = v
	}
	v.note(key)
}

// scan visits every decodable entry of type E in key order. Undecodable entries are
// counted against the table and skipped, so one corrupt value never hides the rest.
func scan[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	s *state,
	visitKey func(key string),
	visit func(e E),
) (err error) {
	name := types.Name[E]()
	prefix := gorp.Prefix[E]()
	it, err := s.db.OpenIterator(kv.IterPrefix(prefix))
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, it.Close()) }()
	for it.First(); it.Valid(); it.Next() {
		if err = ctx.Err(); err != nil {
			return err
		}
		if visitKey != nil {
			visitKey(string(it.Key()[len(prefix):]))
		}
		var e E
		if decodeErr := s.codec.Decode(ctx, it.Value(), &e); decodeErr != nil {
			s.noteDecode(name, strconv.Quote(string(it.Key())))
			continue
		}
		visit(e)
	}
	return errors.Combine(it.Error(), err)
}

// table binds one gorp entry type to the doctor's two scanning passes.
type table struct {
	// name is the gorp type name the table's entries are stored under.
	name string
	// ontologyType is the ontology type the table's entries back. Empty for entries
	// with no ontology presence.
	ontologyType ontology.ResourceType
	// migrations is the chain the binary ships for the table.
	migrations []migrate.Migration
	// collect runs the first pass, recording what other checks reference.
	collect func(ctx context.Context, s *state) error
	// check runs the second pass, emitting findings against the collected state.
	check func(ctx context.Context, s *state) error
}

// tableConfig declares how the doctor reads one gorp table.
type tableConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// migrations is the chain the binary ships for the table.
	migrations []migrate.Migration
	// ontologyID names the resource an entry backs. Nil for entries with no
	// ontology presence.
	ontologyID func(E) ontology.ID
	// collect records what other tables' checks reference.
	collect func(s *state, e E)
	// checkKey validates one stored key, with the table's prefix removed.
	checkKey func(s *state, key string)
	// check emits findings for one entry against the collected state.
	check func(s *state, e E)
}

// newTable builds a table from its config. The first pass always runs, so every table
// reports its undecodable entries even when nothing references it.
func newTable[K gorp.Key, E gorp.Entry[K]](cfg tableConfig[K, E]) table {
	t := table{name: types.Name[E](), migrations: cfg.migrations}
	if cfg.ontologyID != nil {
		var zero E
		t.ontologyType = cfg.ontologyID(zero).Type
	}
	t.collect = func(ctx context.Context, s *state) error {
		return scan[K, E](ctx, s, keyVisitor(s, cfg.checkKey), func(e E) {
			if cfg.ontologyID != nil {
				s.entity(cfg.ontologyID(e))
			}
			if cfg.collect != nil {
				cfg.collect(s, e)
			}
		})
	}
	if cfg.check != nil {
		t.check = func(ctx context.Context, s *state) error {
			return scan[K, E](ctx, s, nil, func(e E) { cfg.check(s, e) })
		}
	}
	return t
}

// keyVisitor binds check to s, returning nil when there is no key check to run.
func keyVisitor(s *state, check func(*state, string)) func(string) {
	if check == nil {
		return nil
	}
	return func(key string) { check(s, key) }
}
