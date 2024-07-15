// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type DependencyList, type EffectCallback, useEffect } from "react";

import { useMemoCompare } from "@/memo";

/* An async version of React.Destructor */
 
export type AsyncDestructor = Promise<(() => void) | void>;

/** An async version of React.EffectCallback */
export type AsyncEffectCallback = () => AsyncDestructor;

/**
 * A version of useEffect that supports async functions and destructors. NOTE: The behavior
 * of this hook hasn't been carefully though out, so it may produce unexpected results.
 * Caveat emptor.
 *
 * @param effect - The async effect callback.
 * @param deps - The dependencies of the effect.
 */
export const useAsyncEffect = (effect: AsyncEffectCallback, deps?: unknown[]): void => {
  useEffect(() => {
    const p = effect();
    return () => {
      p.then((d) => d?.()).catch((e) => console.error(e));
    };
  }, deps);
};

export const useEffectCompare = <D extends DependencyList>(
  cbk: EffectCallback,
  areEqual: (prevDeps: D, nextDeps: D) => boolean,
  deps: D,
): void => {
  const memoDeps = useMemoCompare(() => deps, areEqual, deps);
  useEffect(cbk, [memoDeps]);
};
