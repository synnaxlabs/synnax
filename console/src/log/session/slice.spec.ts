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

import {
  actions,
  reducer,
  SLICE_NAME,
  type SliceState,
  sliceStateZ,
  stateZ,
} from "@/log/session/slice";

const storeWith = (slice: SliceState) =>
  configureStore({
    reducer: { [SLICE_NAME]: reducer },
    preloadedState: { [SLICE_NAME]: slice },
  });

describe("Log Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(sliceStateZ.parse({}));
  });

  describe("create", () => {
    it("should bootstrap session state for the key", () => {
      store.dispatch(actions.create({ key: "log-1" }));
      expect(store.getState()[SLICE_NAME].logs["log-1"]).toEqual(stateZ.parse({}));
    });

    it("should create multiple logs independently", () => {
      store.dispatch(actions.create({ key: "log-1" }));
      store.dispatch(actions.create({ key: "log-2" }));
      expect(Object.keys(store.getState()[SLICE_NAME].logs)).toHaveLength(2);
    });

    it("should not overwrite an existing entry", () => {
      store.dispatch(actions.create({ key: "log-1" }));
      store.dispatch(actions.setActiveToolbarTab({ key: "log-1", tab: "properties" }));
      store.dispatch(actions.create({ key: "log-1" }));
      expect(store.getState()[SLICE_NAME].logs["log-1"].toolbar.activeTab).toBe(
        "properties",
      );
    });
  });

  describe("setActiveToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      store.dispatch(actions.create({ key: "log-1" }));
      store.dispatch(actions.setActiveToolbarTab({ key: "log-1", tab: "properties" }));
      expect(store.getState()[SLICE_NAME].logs["log-1"].toolbar.activeTab).toBe(
        "properties",
      );
    });

    it("should bootstrap state when the log does not yet exist", () => {
      store.dispatch(actions.setActiveToolbarTab({ key: "absent", tab: "properties" }));
      expect(store.getState()[SLICE_NAME].logs.absent.toolbar.activeTab).toBe(
        "properties",
      );
    });
  });

  describe("remove", () => {
    it("should remove a log by key", () => {
      store.dispatch(actions.create({ key: "log-1" }));
      store.dispatch(actions.remove({ keys: ["log-1"] }));
      expect(store.getState()[SLICE_NAME].logs["log-1"]).toBeUndefined();
    });

    it("should remove multiple logs at once", () => {
      store.dispatch(actions.create({ key: "log-1" }));
      store.dispatch(actions.create({ key: "log-2" }));
      store.dispatch(actions.remove({ keys: ["log-1", "log-2"] }));
      expect(Object.keys(store.getState()[SLICE_NAME].logs)).toHaveLength(0);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => stateZ.parse({})).not.toThrow();
    });

    it("should default the toolbar when missing", () => {
      expect(stateZ.parse({}).toolbar.activeTab).toBe("channels");
    });
  });
});
