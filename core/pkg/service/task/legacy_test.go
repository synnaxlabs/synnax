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
	"github.com/synnaxlabs/synnax/pkg/service/opc"
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
		stores = append(stores, MustOpen(opc.OpenService(ctx, opc.ServiceConfig{
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

	// Scan and rack status task configs are created by the driver on rack boot, and
	// arc task configs deploy from Arc; the released Console never exported any of
	// them, so they carry no legacy fixture.
	neverExported := func(t string) bool {
		return strings.HasSuffix(t, "_scan") ||
			t == "ni_scanner" ||
			t == "rack_status" ||
			t == arctask.Type
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
	// must land every field in the typed config store: the legacy rewrite's output
	// must be a subset of the stored record, so a key the new schema does not accept,
	// a mis-renamed key, or a corrupted value fails with the exact path.
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
		},
		Entry(nil, "ethercat_read"),
		Entry(nil, "ethercat_write"),
		Entry(nil, "http_read"),
		Entry(nil, "http_write"),
		Entry(nil, "labjack_read"),
		Entry(nil, "labjack_write"),
		Entry(nil, "labjack_write_cmd_key"),
		Entry(nil, "modbus_read"),
		Entry(nil, "modbus_write"),
		Entry(nil, "ni_analog_read"),
		Entry(nil, "ni_analog_read_config_device"),
		Entry(nil, "ni_analog_write"),
		Entry(nil, "ni_counter_read"),
		Entry(nil, "ni_digital_read"),
		Entry(nil, "ni_digital_write"),
		Entry(nil, "opc_read"),
		Entry(nil, "opc_write"),
		Entry(nil, "pagerduty_alert"),
	)
})
