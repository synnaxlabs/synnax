// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { CurriedGetDefaultMiddleware } from "@reduxjs/toolkit/dist/getDefaultMiddleware";
import { describe, expect, it, vi } from "vitest";

import { configureMiddleware, middleware } from "./middleware";
import { MockRuntime } from "./mock/runtime";

import { initialState, setWindowStage } from "@/state";

const state = {
  drift: initialState,
};

describe("middleware", () => {
  describe("middleware", () => {
    describe("emitting actions", () => {
      it("should emit an action if it hasn't already been emited", () => {
        const store = { getState: () => state, dispatch: vi.fn() };
        const runtime = new MockRuntime(false);
        const mw = middleware(runtime)(store);
        mw((action) => action)({ type: "test" });
        expect(runtime.emissions).toEqual([
          {
            action: { type: "test" },
            emitter: "mock",
          },
        ]);
      });
      it("should not emit an action if it has already been emited", () => {
        const store = { getState: () => state, dispatch: vi.fn() };
        const runtime = new MockRuntime(false);
        const mw = middleware(runtime)(store);
        mw((action) => action)({ type: "DA@test://test" });
        expect(runtime.emissions).toEqual([]);
      });
    });
    describe("'nexting' actions", () => {
      it("should next an action if it has not been emitted by 'self' ", () => {
        const store = { getState: () => state, dispatch: vi.fn() };
        const runtime = new MockRuntime(false);
        const mw = middleware(runtime)(store);
        const next = vi.fn();
        mw(next)({ type: "test" });
        expect(next).toHaveBeenCalledWith({ type: "test" });
      });
      it("should not next an action if it has been emitted by 'self' ", () => {
        const store = { getState: () => state, dispatch: vi.fn() };
        const runtime = new MockRuntime(false);
        const mw = middleware(runtime)(store);
        const next = vi.fn();
        mw(next)({ type: "DA@mock://test" });
        expect(next).not.toHaveBeenCalled();
      });
    });
    describe("key assignment", () => {
      it("should auto-assign a key to a drift action when it isn't present", () => {
        const store = { getState: () => state, dispatch: vi.fn() };
        const runtime = new MockRuntime(false);
        const mw = middleware(runtime)(store);
        const next = vi.fn();
        mw(next)(setWindowStage({ stage: "created" }));
        expect(next).toHaveBeenCalledWith({
          type: "drift/setWindowState",
          payload: {
            key: "mock",
            state: "created",
          },
        });
      });
      it("should not auto-assign a key to a drift action if it has been emitted", () => {
        const store = { getState: () => state, dispatch: vi.fn() };
        const runtime = new MockRuntime(false);
        const mw = middleware(runtime)(store);
        const next = vi.fn();
        mw(next)({
          type: "DA@test://drift/setWindowState",
          payload: {
            key: "mock",
            state: "created",
          },
        });
        expect(next).toHaveBeenCalledWith({
          type: "drift/setWindowState",
          payload: {
            key: "mock",
            state: "created",
          },
        });
      });
    });
  });
  describe("configureMiddleware", () => {
    it("should return a function that returns a middleware when an empty array is provided", () => {
      const runtime = new MockRuntime(true);
      const mwF = configureMiddleware([], runtime);
      expect(typeof mwF).toBe("function");
      expect(mwF([] as unknown as CurriedGetDefaultMiddleware<unknown>).length).toBe(1);
    });
    it("should call a middleware curry function when provided", () => {
      const runtime = new MockRuntime(true);
      const curry = vi.fn();
      const mw = configureMiddleware(() => {
        curry();
        return [];
      }, runtime);
      mw([] as unknown as CurriedGetDefaultMiddleware<unknown>);
      expect(curry).toHaveBeenCalled();
    });
  });
});
