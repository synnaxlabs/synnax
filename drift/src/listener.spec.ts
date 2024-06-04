// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnknownAction } from "@reduxjs/toolkit";
import { describe, expect, it, vi } from "vitest";

import { listen, type StoreDispatch, type StoreStateGetter } from "@/listener";
import { MockRuntime } from "@/mock/runtime";
import { initialState, type StoreState } from "@/state";

const state = {
  drift: initialState,
};

const mockStoreFn = (): StoreStateGetter<StoreState> &
  StoreDispatch<UnknownAction> => ({
  getState: () => state,
  dispatch: vi.fn(),
});

describe("listener", () => {
  describe("listen", () => {
    describe("state forwarding", () => {
      it("should forward it's state upon request if it is the main window", () => {
        const communicator = new MockRuntime(true);
        listen(communicator, mockStoreFn, () => {});
        communicator.subscribeCallback({ emitter: "test", sendState: true });
        expect(communicator.emissions).toEqual([{ state, emitter: "mock" }]);
      });
      it("should not forward it's state upon request if it is not the main window", () => {
        const communicator = new MockRuntime(false);
        listen(communicator, mockStoreFn, () => {});
        communicator.subscribeCallback({ emitter: "test", sendState: true });
        expect(communicator.emissions).toEqual([]);
      });
    });
    describe("initial state receiving", () => {
      it("should resolve the promise upon receiving initial state if it has an undefined store", () => {
        const communicator = new MockRuntime(true);
        const resolve = vi.fn();
        listen(communicator, () => undefined, resolve);
        communicator.subscribeCallback({ emitter: "test", state });
        expect(resolve).toHaveBeenCalledWith(state);
      });
      it("should not resolve the promise upon receiving initial state if it has a defined store", () => {
        const communicator = new MockRuntime(true);
        const resolve = vi.fn();
        listen(communicator, mockStoreFn, resolve);
        communicator.subscribeCallback({ emitter: "test", state });
        expect(resolve).not.toHaveBeenCalled();
      });
    });
    describe("action dispatching", () => {
      it("should dispatch the action if it has a defined store", () => {
        const communicator = new MockRuntime(true);
        const dispatch = vi.fn();
        listen(
          communicator,
          () => ({ getState: () => state, dispatch }),
          () => {},
        );
        communicator.subscribeCallback({ emitter: "test", action: { type: "test" } });
        expect(dispatch).toHaveBeenCalledWith({
          type: "DA@test://test",
        });
      });
      it("should not dispatch the action if it has an undefined store", () => {
        const communicator = new MockRuntime(true);
        const dispatch = vi.fn();
        listen(
          communicator,
          () => undefined,
          () => {},
        );
        communicator.subscribeCallback({ emitter: "test", action: { type: "test" } });
        expect(dispatch).not.toHaveBeenCalled();
      });
    });
  });
});
