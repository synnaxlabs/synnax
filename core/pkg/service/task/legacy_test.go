// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	"github.com/synnaxlabs/synnax/pkg/service/modbus"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktask "github.com/synnaxlabs/synnax/pkg/service/rack/task"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// asFloat normalizes any numeric type to float64. The fixtures decode from JSON as
// float64 while stored configs decode from msgpack as sized integers, so numeric
// leaves compare by value, not type.
func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	return 0, false
}

// legacyDiscarded lists the keys a rewrite drops on purpose, keyed by the task type
// whose released export carried them. The current schema has no field to hold the
// value.
var legacyDiscarded = map[string]set.Set[string]{
	// Only thermocouples ever read pos_chan, and the rewrite folds its value into the
	// port they now read from.
	"labjack_read": set.New("pos_chan"),
	// Digital channels carried a type tag with one possible value.
	"ni_digital_read":  set.New("type"),
	"ni_digital_write": set.New("type"),
}

// legacyVariantDiscarded lists the keys a rewrite drops on purpose from one union
// variant, keyed by the variant's type tag. It scopes a drop that legacyDiscarded
// would apply to every variant of the task type. NI-DAQmx accepts one unit for each
// of the units channels, so the schema carries no units field and the driver passes
// the constant.
var legacyVariantDiscarded = map[string]set.Set[string]{
	// The built-in temperature sensor is not wired to a port. The Driver reads it
	// at a fixed location, so the schema carries no port.
	"ai_temp_builtin": set.New("port"),
	"ai_current":      set.New("units"),
	"ai_current_rms":  set.New("units"),
	"ai_freq_voltage": set.New("units"),
	// The spelling released Drivers accepted, renamed by the rewrite.
	"ai_frequency_voltage":  set.New("units"),
	"ai_microphone":         set.New("units"),
	"ai_resistance":         set.New("units"),
	"ai_strain_gauge":       set.New("units"),
	"ai_voltage":            set.New("units"),
	"ai_voltage_rms":        set.New("units"),
	"ai_voltage_with_excit": set.New("units"),
	"ao_current":            set.New("units"),
	"ao_voltage":            set.New("units"),
}

// The three tables below map a legacy key to where its value lands in the stored
// config. Each is consulted only when the key is absent under its own name, so a type
// that kept the legacy spelling is unaffected.

// legacyRenames maps a legacy key to the stored key holding its value.
var legacyRenames = map[string]string{
	"enabled":     "disabled",
	"data_saving": "data_saving_disabled",
	"subindex":    "sub_index",
	"cmd_key":     "cmd_channel",
	"state_key":   "state_channel",
}

// legacyCollapsed maps a legacy key to the stored key its value folds into. The fold
// changes shape, so only the target's presence is checked.
var legacyCollapsed = map[string]string{
	"cjc_source": "cjc",
	"cjc_val":    "cjc",
	"cjc_port":   "cjc",
}

// legacyLifted maps a legacy config-level key to the list whose elements carry it in
// the stored config. NI v0 stored the analog read device at the config level; the
// rewrite copies it onto every channel.
var legacyLifted = map[string]string{"device": "channels"}

// snakeKey converts a legacy camelCase key to its snake_case stored spelling.
func snakeKey(k string) string {
	isLowerOrDigit := func(c byte) bool {
		return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
	}
	out := make([]byte, 0, len(k)+4)
	for i := range len(k) {
		c := k[i]
		if c < 'A' || c > 'Z' {
			out = append(out, c)
			continue
		}
		if i > 0 && isLowerOrDigit(k[i-1]) {
			out = append(out, '_')
		}
		out = append(out, c+('a'-'A'))
	}
	return string(out)
}

// expectNoDroppedKeys asserts that every key of the legacy fixture reaches the stored
// config, resolving the rewrites that move a value to another key. The subset check
// cannot see a key the rewrite drops: the dropped key is missing from its output and
// from the stored record alike, so both sides agree on nothing.
func expectNoDroppedKeys(path string, legacy, stored any, discarded set.Set[string]) {
	GinkgoHelper()
	switch leg := legacy.(type) {
	case map[string]any:
		// A record whose keys are data becomes a list of pairs, moving each key into
		// the value position of its own element.
		if list, ok := stored.([]any); ok {
			Expect(list).To(HaveLen(len(leg)), "%s: record lost entries", path)
			for k, v := range leg {
				el := MustBeOk(findPair(list, k))
				expectNoDroppedKeys(path+"."+k, v, el["value"], discarded)
			}
			return
		}
		act, ok := stored.(map[string]any)
		Expect(ok).To(BeTrue(), "%s: expected an object, got %T", path, stored)
		for k, v := range leg {
			// A null in a legacy file carried no value; the typed store may omit
			// the field entirely.
			if v == nil {
				continue
			}
			sk := snakeKey(k)
			if av, ok := act[sk]; ok {
				expectNoDroppedKeys(path+"."+k, v, av, discarded)
				continue
			}
			if renamed, ok := legacyRenames[sk]; ok {
				if av, ok := act[renamed]; ok {
					expectNoDroppedKeys(path+"."+k, v, av, discarded)
					continue
				}
			}
			if folded, ok := legacyCollapsed[sk]; ok {
				if _, ok := act[folded]; ok {
					continue
				}
			}
			if discarded.Contains(sk) {
				continue
			}
			if variant, ok := leg["type"].(string); ok &&
				legacyVariantDiscarded[variant].Contains(sk) {
				continue
			}
			liftedInto, lifted := legacyLifted[sk]
			Expect(lifted).To(BeTrue(), "%s: dropped key %q", path, k)
			list, ok := act[liftedInto].([]any)
			Expect(ok).To(BeTrue(), "%s: %q lifted into no list", path, k)
			for i, el := range list {
				Expect(el).To(
					HaveKey(sk), "%s: %q missing from %s[%d]", path, k, liftedInto, i,
				)
			}
		}
	case []any:
		act, ok := stored.([]any)
		Expect(ok).To(BeTrue(), "%s: expected a list, got %T", path, stored)
		Expect(act).To(HaveLen(len(leg)), "%s: list length changed", path)
		for i, v := range leg {
			expectNoDroppedKeys(fmt.Sprintf("%s[%d]", path, i), v, act[i], discarded)
		}
	}
}

// findPair returns the element of list whose record key is key.
func findPair(list []any, key string) (map[string]any, bool) {
	for _, el := range list {
		m, ok := el.(map[string]any)
		if !ok {
			continue
		}
		for _, v := range m {
			if v == key {
				return m, true
			}
		}
	}
	return nil, false
}

// expectSubset asserts that every key path and leaf value in expected survives into
// actual. actual may carry extra keys (applied defaults, minted record keys).
func expectSubset(path string, expected, actual any) {
	GinkgoHelper()
	switch exp := expected.(type) {
	case map[string]any:
		act, ok := actual.(map[string]any)
		Expect(ok).To(BeTrue(), "%s: expected an object, got %T", path, actual)
		for k, v := range exp {
			// A null in a legacy file carried no value; the typed store may omit
			// the field entirely.
			if v == nil {
				continue
			}
			Expect(act).To(HaveKey(k), "%s: missing key %q", path, k)
			expectSubset(path+"."+k, v, act[k])
		}
	case []any:
		act, ok := actual.([]any)
		Expect(ok).To(BeTrue(), "%s: expected a list, got %T", path, actual)
		Expect(act).To(HaveLen(len(exp)), "%s: list length changed", path)
		for i, v := range exp {
			expectSubset(fmt.Sprintf("%s[%d]", path, i), v, act[i])
		}
	default:
		if expF, ok := asFloat(expected); ok {
			actF, ok := asFloat(actual)
			Expect(ok).To(BeTrue(), "%s: expected a number, got %T", path, actual)
			Expect(actF).To(Equal(expF), "%s: value changed", path)
			return
		}
		Expect(actual).To(Equal(expected), "%s: value changed", path)
	}
}

var _ = Describe("Legacy file import", Ordered, ContinueOnFailure, func() {
	var (
		svc     *task.Service
		imexSvc *imex.Service
		configs config.Registry
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               groupSvc,
			HostProvider:        mock.NewStaticHostProvider(1),
			Status:              statusSvc,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))
		var stores []config.Store
		stores = append(
			stores,
			MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{
				DB: db,
			})).Stores()...)
		stores = append(
			stores,
			MustOpen(ethercat.OpenService(ctx, ethercat.ServiceConfig{
				DB: db,
			})).Stores()...)
		stores = append(stores, MustOpen(http.OpenService(ctx, http.ServiceConfig{
			DB: db,
		})).Stores()...)
		stores = append(stores, MustOpen(labjack.OpenService(ctx, labjack.ServiceConfig{
			DB: db,
		})).Stores()...)
		stores = append(stores, MustOpen(modbus.OpenService(ctx, modbus.ServiceConfig{
			DB: db,
		})).Stores()...)
		stores = append(stores, MustOpen(ni.OpenService(ctx, ni.ServiceConfig{
			DB: db,
		})).Stores()...)
		stores = append(stores, MustOpen(opcua.OpenService(ctx, opcua.ServiceConfig{
			DB: db,
		})).Stores()...)
		stores = append(
			stores,
			MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
				DB: db,
			})).Stores()...)
		stores = append(
			stores,
			MustOpen(racktask.OpenService(ctx, racktask.ServiceConfig{
				DB: db,
			})).Stores()...)
		configs = MustSucceed(config.NewRegistry(stores...))
		imexSvc = imex.NewService()
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Rack:     rackSvc,
			Status:   statusSvc,
			Search:   searchIdx,
			ImEx:     imexSvc,
			Configs:  configs,
		}))
	})

	// Scan and rack status tasks are created internal by the driver on rack boot;
	// internal tasks got no ontology resource in released cores, so no released
	// Console surface could export them and they carry no legacy fixture.
	neverExported := func(t string) bool {
		return strings.HasSuffix(t, "_scan") ||
			t == "ni_scanner" ||
			t == "rack_status"
	}

	It("covers every registered config type with a fixture", func() {
		entries := MustSucceed(os.ReadDir(filepath.Join("testdata", "legacy")))
		fixtures := make(set.Set[string], len(entries))
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			fixtures.Add(strings.TrimSuffix(e.Name(), ".json"))
		}
		for _, t := range configs.Types() {
			if neverExported(t) {
				continue
			}
			Expect(fixtures.Contains(string(t))).To(
				BeTrue(), "registered config type %q has no legacy fixture", t,
			)
		}
		for f := range fixtures {
			owned := false
			for _, t := range configs.Types() {
				if strings.HasPrefix(f, string(t)) {
					owned = true
					break
				}
			}
			Expect(owned).To(
				BeTrue(), "fixture %q matches no registered config type", f,
			)
		}
	})

	// Each fixture in testdata/legacy is a frozen copy of what a released Console
	// exported for its task type, with every schema field set to a distinctive
	// non-default value. A type with more than one released shape carries one fixture
	// per shape, suffixed with the shape's distinguishing trait. Importing a fixture
	// must land every field in the typed config store. Two checks cover that: every
	// fixture key reaches the stored record, catching one the rewrite drops, and the
	// rewrite's output is a subset of the stored record, catching a key the new
	// schema does not accept, a mis-renamed key, or a corrupted value.
	DescribeTable("preserves every field of a released Console export",
		func(ctx SpecContext, fixture string) {
			raw := MustSucceed(os.ReadFile(
				filepath.Join("testdata", "legacy", fixture+".json"),
			))
			var env imex.Envelope
			Expect(json.Unmarshal(raw, &env)).To(Succeed())
			Expect(strings.HasPrefix(fixture, env.Type)).To(
				BeTrue(), "fixture %q must be named for its type %q", fixture, env.Type,
			)
			id := MustSucceed(imexSvc.Import(ctx, nil, env, imex.ImportOptions{
				Parent:   ontology.RootID,
				FileName: fixture + ".json",
			}))
			var imported task.Task
			Expect(svc.NewRetrieve().
				Where(task.MatchKeys(uuid.MustParse(id.Key))).
				Entry(&imported).
				Exec(ctx, nil)).To(Succeed())
			Expect(imported.Type).To(Equal(env.Type))
			Expect(imported.Name).To(Equal(fixture))

			var fixtureBody map[string]any
			Expect(json.Unmarshal(raw, &fixtureBody)).To(Succeed())
			delete(fixtureBody, "type")
			expectNoDroppedKeys(
				"config",
				fixtureBody,
				map[string]any(imported.Config),
				legacyDiscarded[env.Type],
			)

			var legacy map[string]any
			Expect(json.Unmarshal(raw, &legacy)).To(Succeed())
			delete(legacy, "type")
			store := MustBeOk(configs.Store(env.Type))
			expected := MustSucceed(store.Normalize(0, legacy))
			expectSubset(
				"config",
				map[string]any(expected),
				map[string]any(imported.Config),
			)

			// A canonical config must pass through the legacy rewrite unchanged, so
			// re-importing an unversioned copy of a current export never rewrites it.
			Expect(store.Normalize(0, imported.Config)).
				To(Equal(imported.Config))

			// The golden file freezes the canonical stored record independently of
			// the rewrite under test: a rewrite edit that silently drops a released
			// field passes the subset check above (the expected side loses the field
			// too) but diffs here. Regenerate deliberately with
			// UPDATE_LEGACY_GOLDENS=1 after reviewing the diff.
			canonical := make(map[string]any, len(imported.Config))
			for k, v := range imported.Config {
				if k != "key" {
					canonical[k] = v
				}
			}
			goldenPath := filepath.Join(
				"testdata",
				"legacy",
				"expected",
				fixture+".json",
			)
			// The trailing newline keeps regenerated goldens Prettier-clean.
			goldenBytes := append(
				MustSucceed(json.MarshalIndent(canonical, "", "  ")), '\n',
			)
			if os.Getenv("UPDATE_LEGACY_GOLDENS") != "" {
				Expect(os.WriteFile(goldenPath, goldenBytes, 0o644)).To(Succeed())
			} else {
				golden := MustSucceed(os.ReadFile(goldenPath))
				Expect(string(goldenBytes)).To(MatchJSON(golden))
			}
		},
		Entry(nil, "arc"),
		Entry(nil, "arc_py"),
		Entry(nil, "ethercat_read"),
		Entry(nil, "ethercat_write"),
		Entry(nil, "http_read"),
		Entry(nil, "http_read_list"),
		Entry(nil, "http_write"),
		Entry(nil, "labjack_read"),
		Entry(nil, "labjack_read_py_no_scale"),
		Entry(nil, "labjack_write"),
		Entry(nil, "labjack_write_cmd_key"),
		Entry(nil, "modbus_read"),
		Entry(nil, "modbus_write"),
		Entry(nil, "ni_analog_read"),
		Entry(nil, "ni_analog_read_config_device"),
		Entry(nil, "ni_analog_read_py"),
		Entry(nil, "ni_analog_write"),
		Entry(nil, "ni_counter_read"),
		Entry(nil, "ni_counter_read_py"),
		Entry(nil, "ni_digital_read"),
		Entry(nil, "ni_digital_write"),
		Entry(nil, "opc_read"),
		Entry(nil, "opc_read_array"),
		Entry(nil, "opc_write"),
		Entry(nil, "pagerduty_alert"),
	)
})
