// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, createAction, type Middleware } from "@reduxjs/toolkit";
import { UnexpectedError } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Window } from "@/layered/session/window";

interface LeftPayload extends Window.OptionalKeyParams {
  key: string;
}

interface ResizePayload extends Window.OptionalKeyParams {
  size: number;
}

const selectLeft = createAction<LeftPayload>("nav/selectLeft");
const resizeLeft = createAction<ResizePayload>("nav/resizeLeft");
const hideAll = createAction<Window.OptionalKeyParams>("nav/hideAll");
const unrelated = createAction<{ value: number }>("other/unrelated");

const driftState = (
  label: string,
  labelKeys: Record<string, string>,
): Drift.StoreState => ({
  drift: { ...Drift.ZERO_SLICE_STATE, label, labelKeys },
});

const RESOLVED = driftState("main", { main: "window-1" });
const UNRESOLVED = driftState("ghost", {});

const run = (
  middleware: Middleware<{}, Drift.StoreState>,
  state: Drift.StoreState,
  action: Action,
) => {
  const next = vi.fn((a: unknown) => a);
  const dispatch = vi.fn();
  const result = middleware({ getState: () => state, dispatch })(next)(action);
  return { next, result };
};

describe("createInjectKeyMiddleware", () => {
  describe("single action creator", () => {
    it("should inject the active window key when the payload has none", () => {
      const middleware = Window.createInjectKeyMiddleware(selectLeft);
      const action = selectLeft({ key: "a" });
      const { next } = run(middleware, RESOLVED, action);
      expect(action.payload.windowKey).toBe("window-1");
      expect(next).toHaveBeenCalledWith(action);
    });

    it("should leave an explicitly provided window key untouched", () => {
      const middleware = Window.createInjectKeyMiddleware(selectLeft);
      const action = selectLeft({ key: "a", windowKey: "explicit" });
      const { next } = run(middleware, RESOLVED, action);
      expect(action.payload.windowKey).toBe("explicit");
      expect(next).toHaveBeenCalledWith(action);
    });

    it("should pass through actions that do not match the creator", () => {
      const middleware = Window.createInjectKeyMiddleware(selectLeft);
      const action = unrelated({ value: 1 });
      const { next } = run(middleware, RESOLVED, action);
      expect(next).toHaveBeenCalledWith(action);
      expect(action.payload).not.toHaveProperty("windowKey");
    });

    it("should drop a matching action when no active window key can be resolved", () => {
      const middleware = Window.createInjectKeyMiddleware(selectLeft);
      const action = selectLeft({ key: "a" });
      const { next, result } = run(middleware, UNRESOLVED, action);
      expect(next).not.toHaveBeenCalled();
      expect(result).toBeUndefined();
      expect(action.payload.windowKey).toBeUndefined();
    });
  });

  describe("array of action creators", () => {
    it("should inject the window key for any of the matched creators", () => {
      const middleware = Window.createInjectKeyMiddleware([
        selectLeft,
        resizeLeft,
        hideAll,
      ]);
      const select = selectLeft({ key: "a" });
      const resize = resizeLeft({ size: 10 });
      const hide = hideAll({});
      run(middleware, RESOLVED, select);
      run(middleware, RESOLVED, resize);
      run(middleware, RESOLVED, hide);
      expect(select.payload.windowKey).toBe("window-1");
      expect(resize.payload.windowKey).toBe("window-1");
      expect(hide.payload.windowKey).toBe("window-1");
    });

    it("should pass through an action matched by none of the creators", () => {
      const middleware = Window.createInjectKeyMiddleware([selectLeft, resizeLeft]);
      const action = hideAll({});
      const { next } = run(middleware, RESOLVED, action);
      expect(next).toHaveBeenCalledWith(action);
      expect(action.payload.windowKey).toBeUndefined();
    });
  });
});

describe("createWithKeyHandler", () => {
  const stateZ = z.object({ count: z.number().default(0) });
  const withKey = Window.createWithKeyHandler(stateZ);

  interface SliceState {
    windows: Record<string, z.output<typeof stateZ>>;
  }

  interface IncrementPayload extends Window.OptionalKeyParams {
    by: number;
  }

  const increment = createAction<IncrementPayload>("nav/increment");

  it("should throw an UnexpectedError when the payload has no window key", () => {
    const handler = withKey<Window.OptionalKeyParams, SliceState>(() => {});
    const state: SliceState = { windows: {} };
    try {
      handler(state, { type: "nav/increment", payload: {} });
      expect.unreachable("handler should have thrown");
    } catch (e) {
      expect(UnexpectedError.matches(e)).toBe(true);
    }
  });

  it("should initialize a fresh window state from the schema when absent", () => {
    const handler = withKey<Window.OptionalKeyParams, SliceState>((s) => {
      s.count += 1;
    });
    const state: SliceState = { windows: {} };
    handler(state, { type: "nav/increment", payload: { windowKey: "w-1" } });
    expect(state.windows["w-1"]).toEqual({ count: 1 });
  });

  it("should reuse and mutate an existing window state", () => {
    const handler = withKey<Window.OptionalKeyParams, SliceState>((s) => {
      s.count += 1;
    });
    const existing = { count: 5 };
    const state: SliceState = { windows: { "w-1": existing } };
    handler(state, { type: "nav/increment", payload: { windowKey: "w-1" } });
    expect(state.windows["w-1"]).toBe(existing);
    expect(existing.count).toBe(6);
  });

  it("should forward the resolved window state and action to the handler", () => {
    const handler = withKey<IncrementPayload, SliceState>((s, action) => {
      s.count += action.payload.by;
    });
    const state: SliceState = { windows: {} };
    handler(state, increment({ windowKey: "w-1", by: 3 }));
    expect(state.windows["w-1"]).toEqual({ count: 3 });
  });
});
