// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos as alamosLib, type Instrumentation } from "@synnaxlabs/alamos";
import { UnexpectedError } from "@synnaxlabs/client";

import { aether } from "@/aether/aether";
import { type state } from "@/state";

/** Key of the synthetic root mounted by {@link createDriver}. All driver paths begin
 * with this segment. */
export const ROOT_KEY = "root";

/** Low-level handle over a worker-side aether tree. Drives state updates and deletes
 * against a real {@link aether.Root} backed by a mock comms pair, with no knowledge of
 * any particular provider. Higher-level Synnax test utilities build their provider
 * stacks on top of this. */
export interface Driver {
  /** Root of the worker-side tree. */
  root: aether.Root;
  /** Worker end of the mock comms pair. */
  workerSide: aether.WorkerComms;
  /** Main end of the mock comms pair, for bridging a React tree into this driver. */
  mainSide: aether.MainComms;
  /** Create or replace the component at `path`, constructing it from the registry if it
   * does not yet exist. `path` is absolute and begins with {@link ROOT_KEY}. */
  update(path: readonly string[], type: string, state: state.State): void;
  /** Delete the component at `path` and its descendants. */
  delete(path: readonly string[]): void;
  /** Look up a mounted component by absolute path. Throws if no component exists there. */
  find<T extends aether.Component = aether.Component>(path: readonly string[]): T;
}

/**
 * Create a {@link Driver} over a fresh worker-side aether tree using the given registry.
 *
 * The driver constructs components lazily through the registry the same way production
 * does — the caller never builds `parentCtxValues` by hand, so every level runs its real
 * `afterUpdate` lifecycle and propagates context normally.
 */
export const createDriver = (
  registry: aether.ComponentRegistry,
  instrumentation: Instrumentation = alamosLib.Instrumentation.NOOP,
): Driver => {
  const [workerSide, mainSide] = aether.createMockPair();
  const root = aether.render({ worker: workerSide, registry, instrumentation });

  const update = (
    path: readonly string[],
    type: string,
    stateValue: state.State,
  ): void => {
    root._updateState({
      path,
      type,
      state: stateValue,
      create: (parentCtxValues) => {
        const Constructor = registry[type];
        if (Constructor == null)
          throw new UnexpectedError(`[aetherTest] type '${type}' not in registry`);
        return new Constructor({
          key: path[path.length - 1],
          type,
          sender: workerSide,
          instrumentation,
          parentCtxValues,
        });
      },
    });
  };

  const find = <T extends aether.Component = aether.Component>(
    path: readonly string[],
  ): T => {
    const found = root.findChildAtPath([...path.slice(1)]);
    if (found == null)
      throw new UnexpectedError(
        `[aetherTest] component at ${path.join(".")} not found`,
      );
    return found as T;
  };

  return {
    root,
    workerSide,
    mainSide,
    update,
    delete: (path) => root._delete(path),
    find,
  };
};
