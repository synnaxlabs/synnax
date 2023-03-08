// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { selectWindow } from "./selectors";
import {
  closeWindow,
  completeProcess,
  createWindow,
  DriftAction,
  DriftState,
  initialState,
  reducer,
  registerProcess,
  setWindowLabel,
  setWindowStage,
} from "./state";

const reduce = (initialState: DriftState, ...actions: DriftAction[]): DriftState =>
  actions.reduce((state, action) => reducer(state, action), initialState);

describe("state", () => {
  describe("slice", () => {
    describe("setWindowKey", () => {
      it("sets the key", () => {
        const label = "key";
        const state = reduce(initialState, setWindowLabel({ label }));
        expect(state.label).toBe(label);
      });
    });
    describe("createWindow", () => {
      it("should add a widnow to state", () => {
        const key = "key";
        const state = reduce(initialState, createWindow({ key }));
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.stage).toBe("creating");
        expect(win?.processCount).toBe(0);
      });
    });
    describe("setWindowState", () => {
      it("should set the state of a window", () => {
        const key = "key";
        const state = reduce(
          initialState,
          createWindow({ key }),
          setWindowStage({ stage: "closed", key })
        );
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.stage).toBe("closed");
      });
    });
    describe("closeWindow", () => {
      it("should set the state of the window to closing", () => {
        const key = "key";
        const state = reduce(initialState, createWindow({ key }), closeWindow({ key }));
        const win = state.windows[key];
        expect(win).toBeDefined();
        expect(win?.stage).toBe("closing");
      });
    });
    describe("setWindowStage", () => {
      it("should set the window stage", () => {
        const key = "key";
        const state = reduce(
          initialState,
          createWindow({ key }),
          closeWindow({ key }),
          setWindowStage({ stage: "closed", key })
        );
        const win = selectWindow({ drift: state }, key);
        expect(win).toBeDefined();
      });
    });
    describe("registerProcess", () => {
      it("should increment the process count", () => {
        const key = "key";
        const state = reduce(
          initialState,
          createWindow({ key }),
          registerProcess({ key })
        );
        const win = selectWindow({ drift: state }, key);
        expect(win).toBeDefined();
        expect(win?.processCount).toBe(1);
      });
    });
    describe("completeProcess", () => {
      it("should decrement the process count", () => {
        const key = "key";
        const preState = reduce(
          initialState,
          createWindow({ key }),
          registerProcess({ key }),
          registerProcess({ key })
        );
        const state = reducer(preState, completeProcess({ key }));
        const win = selectWindow({ drift: state }, key);
        expect(win).toBeDefined();
        expect(win?.processCount).toBe(1);
      });
    });
  });
});
