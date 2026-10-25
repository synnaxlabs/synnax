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

import { Aether } from "@/aether";
import { input } from "@/vis/input/aether";

export interface UseProps extends Pick<input.State, "sink" | "source"> {
  aetherKey: string;
}

export interface UseReturn {
  set: (value: string) => void;
  /** value is the latest string the source holds, or null before the first sample. */
  value: string | null;
}

export const use = ({ aetherKey, sink, source }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ sink, source });
  const { setState, methods, subscribe, getSnapshot } = Aether.useLifecycle({
    aetherKey,
    type: input.Input.TYPE,
    schema: input.stateZ,
    initialState: memoProps,
    methods: input.methodsZ,
  });
  // Reads only the value, so pushing props into the state does not re-render.
  const value = useSyncExternalStore(subscribe, () => getSnapshot().value);

  useEffect(() => {
    setState((state) => ({ ...state, ...memoProps }));
  }, [memoProps, setState]);

  return { set: methods.set, value };
};
