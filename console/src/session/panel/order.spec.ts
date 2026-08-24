// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Panel } from "@/session/panel";

const createStore = () =>
  configureStore({ reducer: { [Panel.ORDER_SLICE_NAME]: Panel.orderReducer } });

type TestState = ReturnType<ReturnType<typeof createStore>["getState"]>;

// The order is shared by every window, so it needs no window bookkeeping to drive it.
const run = (...actions: Panel.OrderAction[]): TestState => {
  const store = createStore();
  actions.forEach((action) => store.dispatch(action));
  return store.getState();
};

describe("Panel order", () => {
  describe("reconcileOrder", () => {
    const [A, B, C] = Array.from({ length: 3 }, () => uuid.create());

    it("should materialize the order name-sorted on first sight", () => {
      const state = run(
        Panel.reconcileOrder({
          panels: [
            { key: A, name: "Panel 10" },
            { key: B, name: "Panel 2" },
            { key: C, name: "Avionics" },
          ],
        }),
      );
      expect(Panel.selectOrder(state)).toEqual([C, B, A]);
    });

    it("should append new panels name-sorted without moving existing ones", () => {
      const state = run(
        Panel.reconcileOrder({ panels: [{ key: B, name: "Zulu" }] }),
        Panel.reconcileOrder({
          panels: [
            { key: B, name: "Zulu" },
            { key: C, name: "Bravo" },
            { key: A, name: "Alpha" },
          ],
        }),
      );
      expect(Panel.selectOrder(state)).toEqual([B, A, C]);
    });

    it("should prune panels missing from the membership", () => {
      const state = run(
        Panel.reconcileOrder({
          panels: [
            { key: A, name: "Alpha" },
            { key: B, name: "Bravo" },
          ],
        }),
        Panel.reconcileOrder({ panels: [{ key: B, name: "Bravo" }] }),
      );
      expect(Panel.selectOrder(state)).toEqual([B]);
    });

    it("should never move an existing panel on a rename", () => {
      const state = run(
        Panel.reconcileOrder({
          panels: [
            { key: A, name: "Alpha" },
            { key: B, name: "Bravo" },
          ],
        }),
        Panel.reconcileOrder({
          panels: [
            { key: A, name: "Zulu" },
            { key: B, name: "Bravo" },
          ],
        }),
      );
      expect(Panel.selectOrder(state)).toEqual([A, B]);
    });

    // Cross-window echoes re-apply against converged state, so an in-invariant
    // membership must not churn the slice's reference.
    it("should leave a converged order referentially unchanged", () => {
      const store = createStore();
      const panels = [
        { key: A, name: "Alpha" },
        { key: B, name: "Bravo" },
      ];
      store.dispatch(Panel.reconcileOrder({ panels }));
      const before = store.getState()[Panel.ORDER_SLICE_NAME];
      store.dispatch(Panel.reconcileOrder({ panels }));
      expect(store.getState()[Panel.ORDER_SLICE_NAME]).toBe(before);
    });
  });

  describe("reorder", () => {
    const [A, B, C] = Array.from({ length: 3 }, () => uuid.create());
    const seed = Panel.reconcileOrder({
      panels: [
        { key: A, name: "Alpha" },
        { key: B, name: "Bravo" },
        { key: C, name: "Charlie" },
      ],
    });

    // The insertion index counts the moved panel itself, so moving to the last
    // slot arrives as the strip length.
    it("should move a panel toward the end", () => {
      const state = run(seed, Panel.reorder({ key: A, index: 3 }));
      expect(Panel.selectOrder(state)).toEqual([B, C, A]);
    });

    it("should move a panel toward the start", () => {
      const state = run(seed, Panel.reorder({ key: C, index: 0 }));
      expect(Panel.selectOrder(state)).toEqual([C, A, B]);
    });

    // Both slots adjacent to the panel's own pill resolve to its current
    // position, so a drop back in place must not churn the reference.
    it("should leave a drop in place referentially unchanged", () => {
      const store = createStore();
      store.dispatch(seed);
      const before = store.getState()[Panel.ORDER_SLICE_NAME];
      store.dispatch(Panel.reorder({ key: B, index: 1 }));
      store.dispatch(Panel.reorder({ key: B, index: 2 }));
      expect(store.getState()[Panel.ORDER_SLICE_NAME]).toBe(before);
    });

    it("should insert a not-yet-reconciled panel at the index", () => {
      const other = uuid.create();
      const state = run(seed, Panel.reorder({ key: other, index: 1 }));
      expect(Panel.selectOrder(state)).toEqual([A, other, B, C]);
    });

    it("should clamp an out-of-range index to the end", () => {
      const state = run(seed, Panel.reorder({ key: A, index: 10 }));
      expect(Panel.selectOrder(state)).toEqual([B, C, A]);
    });
  });
});
