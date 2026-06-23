// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { beforeEach, describe, expect, it } from "vitest";

import { Log } from "@/layered/session/log";

const storeWith = (slice: Log.SliceState) =>
  configureStore({
    reducer: { [Log.SLICE_NAME]: Log.reducer },
    preloadedState: { [Log.SLICE_NAME]: slice },
  });

const KEY = "log-1";

describe("Log Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Log.ZERO_SLICE_STATE);
  });

  const select = <R>(
    selector: (params: Log.KeyedSelectorParams) => R,
    key: string = KEY,
  ): R => selector({ state: store.getState(), key });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      store.dispatch(Log.internalCreate({ key: KEY }));
      expect(select(Log.selectState)).toEqual(Log.ZERO_STATE);
    });

    it("should create multiple logs independently", () => {
      store.dispatch(Log.internalCreate({ key: "log-1" }));
      store.dispatch(Log.internalCreate({ key: "log-2" }));
      expect(Object.keys(Log.selectSliceState(store.getState()).logs)).toHaveLength(2);
    });

    it("should not overwrite an existing entry", () => {
      store.dispatch(Log.internalCreate({ key: KEY }));
      store.dispatch(Log.setActiveToolbarTab({ key: KEY, tab: "properties" }));
      store.dispatch(Log.internalCreate({ key: KEY }));
      expect(select(Log.selectActiveToolbarTab)).toBe("properties");
    });
  });

  describe("setActiveToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      store.dispatch(Log.internalCreate({ key: KEY }));
      store.dispatch(Log.setActiveToolbarTab({ key: KEY, tab: "properties" }));
      expect(select(Log.selectActiveToolbarTab)).toBe("properties");
    });

    it("should provision state on first action for an unknown key", () => {
      store.dispatch(Log.setActiveToolbarTab({ key: "absent", tab: "properties" }));
      expect(select(Log.selectActiveToolbarTab, "absent")).toBe("properties");
    });
  });

  describe("remove", () => {
    it("should remove a log by key", () => {
      store.dispatch(Log.internalCreate({ key: KEY }));
      store.dispatch(Log.remove({ keys: [KEY] }));
      expect(select(Log.selectExists)).toBe(false);
    });

    it("should remove multiple logs at once", () => {
      store.dispatch(Log.internalCreate({ key: "log-1" }));
      store.dispatch(Log.internalCreate({ key: "log-2" }));
      store.dispatch(Log.remove({ keys: ["log-1", "log-2"] }));
      expect(Object.keys(Log.selectSliceState(store.getState()).logs)).toHaveLength(0);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => Log.stateZ.parse(Log.ZERO_STATE)).not.toThrow();
    });

    it("should default the toolbar when missing", () => {
      const { toolbar: _toolbar, ...withoutToolbar } = Log.ZERO_STATE;
      expect(Log.stateZ.parse(withoutToolbar).toolbar).toEqual(Log.ZERO_STATE.toolbar);
    });

    it("should reject an incorrect version", () => {
      expect(() => Log.stateZ.parse({ ...Log.ZERO_STATE, version: "1.0.0" })).toThrow();
    });
  });
});
