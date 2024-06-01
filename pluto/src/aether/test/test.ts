// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type Mock,vi } from "vitest";

import { aether } from "@/aether/aether";

interface TestLeaf<T extends aether.Component> {
  component: T;
  setState: (state: any) => Promise<void>;
  delete: () => Promise<void>;
  stateChange: Mock;
}

export const render = async <T extends aether.Component>(
  constructor: aether.ComponentConstructor,
  initialState: any,
  setOnContext: (ctx: aether.Context) => void,
): Promise<TestLeaf<T>> => {
  const MockSender = { send: vi.fn() };

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
  await component.internalUpdate(update);

  return {
    component,
    setState: async (state: any) =>
      await component.internalUpdate({ ...update, state }),
    delete: async () => await component.internalDelete(update.path),
    stateChange: MockSender.send,
  };
};
