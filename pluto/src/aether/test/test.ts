// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { vi, type Mock } from "vitest";

import { aether } from "@/aether/aether";

interface TestLeaf<T extends aether.Component> {
  component: T;
  setState: (state: any) => void;
  delete: () => void;
  stateChange: Mock;
}

export const render = <T extends aether.Component>(
  constructor: aether.ComponentConstructor,
  initialState: any,
  setOnContext: (ctx: aether.Context) => void,
): TestLeaf<T> => {
  const MockSender = {
    send: vi.fn(),
  };

  const ctx = new aether.Context(MockSender, {});
  setOnContext(ctx);

  const update: aether.Update = {
    ctx,
    variant: "state",
    type: "test",
    path: ["test"],
    state: initialState,
    instrumentation: alamos.NOOP,
  };

  const component = new constructor(update) as T;
  component.internalUpdate(update);

  return {
    component,
    setState: (state: any) => {
      component.internalUpdate({
        ...update,
        state,
      });
    },
    delete: () => {
      component.internalDelete(update.path);
    },
    stateChange: MockSender.send,
  };
};
