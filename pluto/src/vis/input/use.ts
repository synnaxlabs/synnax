// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect } from "react";

import { Aether } from "@/aether";
import { useMemoDeepEqual } from "@/memo";
import { input } from "@/vis/input/aether";

export interface UseProps extends Pick<input.State, "sink"> {
  aetherKey: string;
}

export interface UseReturn {
  set: (value: string) => void;
}

export const use = ({ aetherKey, sink }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ sink });
  const [, , setState] = Aether.use({
    aetherKey,
    type: input.Input.TYPE,
    schema: input.stateZ,
    initialState: { trigger: 0, ...memoProps },
  });

  useEffect(() => {
    setState((state) => ({ ...state, ...memoProps }));
  }, [memoProps, setState]);

  const handleSet = useCallback(
    (value: string) =>
      setState((state) => ({
        ...state,
        trigger: state.trigger + 1,
        command: value,
      })),
    [setState],
  );

  return { set: handleSet };
};
