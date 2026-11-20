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
import { stringValue } from "@/vis/stringValue/aether";

export interface UseProps extends Omit<z.input<typeof stringValue.stateZ>, "stale"> {
  aetherKey: string;
}

export interface UseReturn {
  stale: boolean;
}

/**
 * use draws the latest value of a string source into a box on the canvas. The worker
 * draws each sample, so a sample never re-renders the calling component.
 */
export const use = ({ aetherKey, ...rest }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual(rest);
  const { setState, subscribe, getSnapshot } = Aether.useLifecycle({
    aetherKey,
    type: stringValue.StringValue.TYPE,
    schema: stringValue.stateZ,
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
