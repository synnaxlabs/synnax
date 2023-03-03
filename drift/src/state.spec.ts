// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  closeWindow,
  completeProcess,
  createWindow,
  initialState,
  reducer,
  registerProcess,
  setWindowKey,
  setWindowStage,
} from "./state";

describe("state", () => {
  describe("slice", () => {
    describe("setWindowKey", () => {
      it("sets the key", () => {
        const key = "key";
        const state = reducer(initialState, setWindowKey({ key }));
        expect(state.key).toBe(key);
      });
    });
    describe("createWindow", () => {
      it("should add a widnow to state", () => {
        const key = "key";
        const state = reducer(initialState, createWindow({ key }));
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.stage).toBe("creating");
        expect(win?.processCount).toBe(0);
      });
    });
    describe("setWindowState", () => {
      it("should set the state of a window", () => {
        const key = "key";
        const state = reducer(
          reducer(initialState, createWindow({ key })),
          setWindowStage({ stage: "closed", key })
        );
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.stage).toBe("closed");
      });
    });
    describe("closeWindow", () => {
      it("should set the state of a window to closing", () => {
        const key = "key";
        const state = reducer(
          reducer(initialState, createWindow({ key })),
          closeWindow({ key })
        );
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.stage).toBe("closed");
      });
    });
    describe("registerProcess", () => {
      it("should increment the process count", () => {
        const key = "key";
        const state = reducer(
          reducer(initialState, createWindow({ key })),
          registerProcess({ key })
        );
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.processCount).toBe(1);
      });
    });
    describe("completeProcess", () => {
      it("should decrement the process count", () => {
        const key = "key";
        const preState = reducer(
          reducer(
            reducer(initialState, createWindow({ key })),
            registerProcess({ key })
          ),
          registerProcess({ key })
        );
        const state = reducer(preState, completeProcess({ key }));
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.processCount).toBe(1);
      });
      it("should close the window if the process count is 0", () => {
        const key = "key";
        const preState = reducer(
          reducer(
            reducer(initialState, createWindow({ key })),
            registerProcess({ key })
          ),
          registerProcess({ key })
        );
        const state = reducer(
          reducer(reducer(preState, closeWindow({ key })), completeProcess({ key })),
          completeProcess({ key })
        );
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.stage).toBe("closed");
      });
    });
  });
});
