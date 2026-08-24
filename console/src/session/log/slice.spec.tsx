// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Log as PLog } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
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

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PLog.Scope.Provider value={key}>{children}</PLog.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("Log Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Log.ZERO_SLICE_STATE);
  });

  const renderGetter = <G,>(useGetter: () => G, key: string = KEY): G =>
    renderHook(() => useGetter(), { wrapper: wrapperFor(store, key) }).result.current;

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
      const getState = renderGetter(Log.useGetState);
      act(() => void store.dispatch(Log.create({ key: KEY })));
      expect(getState()).toEqual(Log.ZERO_STATE);
    });

    it("should create multiple logs independently", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(
          Log.create({ key: "log-1", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.create({ key: "log-2" }));
      });
      expect(getState({ key: "log-1" }).toolbar.selectedTab).toBe("properties");
      expect(getState({ key: "log-2" }).toolbar.selectedTab).toBe("channels");
    });

    it("should not overwrite an existing entry", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setSelectedToolbarTab({ key: KEY, tab: "properties" }));
        store.dispatch(Log.create({ key: KEY }));
      });
      expect(getState().toolbar.selectedTab).toBe("properties");
    });
  });

  describe("setActiveToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setSelectedToolbarTab({ key: KEY, tab: "properties" }));
      });
      expect(getState().toolbar.selectedTab).toBe("properties");
    });

    it("should provision state on first action for an unknown key", () => {
      const getState = renderGetter(Log.useGetState, "absent");
      act(
        () =>
          void store.dispatch(
            Log.setSelectedToolbarTab({ key: "absent", tab: "properties" }),
          ),
      );
      expect(getState().toolbar.selectedTab).toBe("properties");
    });
  });

  describe("setHold", () => {
    it("should set the hold state", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setHold({ key: KEY, hold: true }));
      });
      expect(getState().hold).toBe(true);
    });

    it("should toggle the hold state when hold is omitted", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(Log.create({ key: KEY }));
        store.dispatch(Log.setHold({ key: KEY }));
      });
      expect(getState().hold).toBe(true);
      act(() => void store.dispatch(Log.setHold({ key: KEY })));
      expect(getState().hold).toBe(false);
    });

    it("should provision state on first action for an unknown key", () => {
      const getState = renderGetter(Log.useGetState, "absent");
      act(() => void store.dispatch(Log.setHold({ key: "absent", hold: true })));
      expect(getState().hold).toBe(true);
    });
  });

  describe("remove", () => {
    it("should remove a log by key", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(
          Log.create({ key: KEY, toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.remove({ keys: [KEY] }));
      });
      expect(getState()).toEqual(Log.ZERO_STATE);
    });

    it("should remove multiple logs at once", () => {
      const getState = renderGetter(Log.useGetState);
      act(() => {
        store.dispatch(
          Log.create({ key: "log-1", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(
          Log.create({ key: "log-2", toolbar: { selectedTab: "properties" } }),
        );
        store.dispatch(Log.remove({ keys: ["log-1", "log-2"] }));
      });
      expect(getState({ key: "log-1" })).toEqual(Log.ZERO_STATE);
      expect(getState({ key: "log-2" })).toEqual(Log.ZERO_STATE);
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

    it("should reject an incorrect slice version", () => {
      expect(() =>
        Log.sliceStateZ.parse({ ...Log.ZERO_SLICE_STATE, version: 1 }),
      ).toThrow();
    });
  });
});
