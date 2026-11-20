// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { useEffect, useSyncExternalStore } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { scale } from "@/vis/scale/aether";

export const gutter = scale.gutter;
export const levelColorFallback = scale.levelColorFallback;
export const axisColorFallback = scale.axisColorFallback;
export const textColorFallback = scale.textColorFallback;

export interface UseProps extends Omit<z.input<typeof scale.Scale.z>, "stale"> {
  aetherKey: string;
}

export interface UseReturn {
  stale: boolean;
}

export const use = ({ aetherKey, ...rest }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual(rest);
  const { setState, subscribe, getSnapshot } = Aether.useLifecycle({
    aetherKey,
    type: scale.Scale.TYPE,
    schema: scale.Scale.z,
    initialState: memoProps,
  });
  useEffect(
    () => setState((state) => ({ ...state, ...memoProps })),
    [memoProps, setState],
  );
  // Selects only the staleness, so a state push re-renders only on a transition.
  const stale = useSyncExternalStore(subscribe, () => getSnapshot().stale);
  return { stale };
};
