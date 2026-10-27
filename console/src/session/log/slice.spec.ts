// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Log } from "@/session/log";
import { createSliceStore, documentIn } from "@/session/window/testutil";

const storeWith = (slice: Log.SliceState) =>
  createSliceStore({
    name: Log.SLICE_NAME,
    reducer: Log.reducer,
    preloadedState: slice,
    middleware: Log.MIDDLEWARE,
  });

const KEY = "log-1";
const AUX_WINDOW = "aux-window";

describe("Log Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Log.ZERO_SLICE_STATE);
  });

  const docOf = (key: string = KEY) =>
    documentIn(store.getState()[Log.SLICE_NAME], key);

  // A window is a viewport: an edit in one window must not reach another window's
  // view of the same document.
  it("should keep each window's view of a document independent", () => {
    store.dispatch(Log.create({ key: KEY }));
    store.dispatch(Log.create({ key: KEY, windowKey: AUX_WINDOW }));
    store.dispatch(Log.setHold({ key: KEY, hold: true, windowKey: AUX_WINDOW }));
    const slice = store.getState()[Log.SLICE_NAME];
    expect(documentIn(slice, KEY, AUX_WINDOW)?.hold).toBe(true);
    expect(documentIn(slice, KEY)?.hold).toBe(false);
  });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      act(() => void store.dispatch(Log.create({ key: KEY })));
      expect(docOf()).toEqual(Log.ZERO_STATE);
    });

    it("should create multiple logs independently", () => {
      act(() => {
        store.dispatch(
          Log.create({ key: "log-1", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.create({ key: "log-2" }));
      });
      expect(docOf("log-1")?.toolbar.selectedTab).toBe("properties");
      expect(docOf("log-2")?.toolbar.selectedTab).toBe("channels");
    });

    it("should not overwrite an existing entry", () => {
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setSelectedToolbarTab({ key: KEY, tab: "properties" }));
        store.dispatch(Log.create({ key: KEY }));
      });
      expect(docOf()?.toolbar.selectedTab).toBe("properties");
    });
  });

  describe("setActiveToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setSelectedToolbarTab({ key: KEY, tab: "properties" }));
      });
      expect(docOf()?.toolbar.selectedTab).toBe("properties");
    });

    it("should provision state on first action for an unknown key", () => {
      act(
        () =>
          void store.dispatch(
            Log.setSelectedToolbarTab({ key: "absent", tab: "properties" }),
          ),
      );
      expect(docOf("absent")?.toolbar.selectedTab).toBe("properties");
    });
  });

  describe("setHold", () => {
    it("should set the hold state", () => {
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setHold({ key: KEY, hold: true }));
      });
      expect(docOf()?.hold).toBe(true);
    });

    it("should toggle the hold state when hold is omitted", () => {
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setHold({ key: KEY }));
      });
      expect(docOf()?.hold).toBe(true);
      act(() => void store.dispatch(Log.setHold({ key: KEY })));
      expect(docOf()?.hold).toBe(false);
    });

    it("should provision state on first action for an unknown key", () => {
      act(() => void store.dispatch(Log.setHold({ key: "absent", hold: true })));
      expect(docOf("absent")?.hold).toBe(true);
    });
  });

  describe("remove", () => {
    it("should remove a log by key", () => {
      act(() => {
        store.dispatch(
          Log.create({ key: KEY, toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.remove({ keys: [KEY] }));
      });
      expect(docOf()).toBeUndefined();
    });

    it("should remove multiple logs at once", () => {
      act(() => {
        store.dispatch(
          Log.create({ key: "log-1", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(
          Log.create({ key: "log-2", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.remove({ keys: ["log-1", "log-2"] }));
      });
      expect(docOf("log-1")).toBeUndefined();
      expect(docOf("log-2")).toBeUndefined();
    });
  });

  describe("sliceStateZ schema", () => {
    it("should reject an incorrect slice version", () => {
      expect(() =>
        Log.sliceStateZ.parse({ ...Log.ZERO_SLICE_STATE, version: 1 }),
      ).toThrow();
    });
  });
});
