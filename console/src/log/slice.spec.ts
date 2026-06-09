// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { log } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it } from "vitest";

import {
  actions,
  type PendingUpload,
  reducer,
  SLICE_NAME,
  type SliceState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/log/slice";
import { stateZ } from "@/log/types";

const PENDING: PendingUpload = log.logZ.omit({ name: true }).parse({
  key: "11111111-1111-4111-8111-111111111111",
  channels: [log.channelEntryZ.parse({ channel: 42, color: color.ZERO })],
});

const storeWith = (slice: SliceState) =>
  configureStore({
    reducer: { [SLICE_NAME]: reducer },
    preloadedState: { [SLICE_NAME]: slice },
  });

describe("Log Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(ZERO_SLICE_STATE);
  });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      store.dispatch(actions.create({ key: "log-1" }));
      expect(store.getState()[SLICE_NAME].logs["log-1"]).toEqual({
        ...ZERO_STATE,
        key: "log-1",
      });
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

    it("should be a no-op when the log does not exist", () => {
      store.dispatch(actions.setActiveToolbarTab({ key: "absent", tab: "properties" }));
      expect(store.getState()[SLICE_NAME].logs.absent).toBeUndefined();
    });
  });

  describe("clearPendingUpload", () => {
    it("should clear a pending upload", () => {
      const seeded = storeWith({
        ...ZERO_SLICE_STATE,
        logs: { "log-1": { ...ZERO_STATE, key: "log-1", pendingUpload: PENDING } },
      });
      seeded.dispatch(actions.clearPendingUpload({ key: "log-1" }));
      expect(seeded.getState()[SLICE_NAME].logs["log-1"].pendingUpload).toBeUndefined();
    });

    it("should be a no-op when the log does not exist", () => {
      store.dispatch(actions.clearPendingUpload({ key: "absent" }));
      expect(store.getState()[SLICE_NAME].logs.absent).toBeUndefined();
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
      expect(() => stateZ.parse(ZERO_STATE)).not.toThrow();
    });

    it("should accept a state with a pending upload", () => {
      expect(() =>
        stateZ.parse({ ...ZERO_STATE, key: "log-1", pendingUpload: PENDING }),
      ).not.toThrow();
    });

    it("should default the toolbar when missing", () => {
      const { toolbar: _toolbar, ...withoutToolbar } = ZERO_STATE;
      expect(stateZ.parse(withoutToolbar).toolbar).toEqual(ZERO_STATE.toolbar);
    });

    it("should reject an incorrect version", () => {
      expect(() => stateZ.parse({ ...ZERO_STATE, version: "1.0.0" })).toThrow();
    });
  });
});
