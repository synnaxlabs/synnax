// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useMemoDeepEqual } from "@/memo";
import { setpoint } from "@/vis/setpoint/aether";

export interface UseProps extends Pick<
  z.input<typeof setpoint.stateZ>,
  "source" | "sink"
> {
  aetherKey: string;
}

export interface UseReturn extends Pick<z.infer<typeof setpoint.stateZ>, "value"> {
  set: (value: number) => void;
}

export const use = ({ aetherKey, source, sink }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ source, sink });
  const [, { value }, setState] = Aether.use({
    aetherKey,
    type: setpoint.Setpoint.TYPE,
    schema: setpoint.stateZ,
    initialState: {
      trigger: 0,
      value: 0,
      ...memoProps,
    },
  });

  useEffect(() => {
    setState((state) => ({ ...state, ...memoProps }));
  }, [memoProps, setState]);

  const handleSet = useCallback(
    (value: number) =>
      setState((state) => ({
        ...state,
        trigger: state.trigger + 1,
        command: value,
      })),
    [setState],
  );

  return { set: handleSet, value };
};
