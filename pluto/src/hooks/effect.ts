// Copyright 2025 Synnax Labs, Inc.
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

export type AsyncDestructor = (() => Promise<void>) | (() => void) | void;

/** An async version of React.EffectCallback */
export type AsyncEffectCallback = () => Promise<AsyncDestructor>;

export const useEffectCompare = <D extends DependencyList>(
  cbk: EffectCallback,
  areEqual: (prevDeps: D, nextDeps: D) => boolean,
  deps: D,
): void => {
  const memoDeps = useMemoCompare(() => deps, areEqual, deps);
  useEffect(cbk, [memoDeps]);
};
