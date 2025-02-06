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
import { useMemoDeepEqualProps } from "@/memo";
import { toggle } from "@/vis/toggle/aether";

export interface UseProps
  extends Pick<z.input<typeof toggle.toggleStateZ>, "source" | "sink"> {
  aetherKey: string;
}

export interface UseReturn
  extends Pick<z.output<typeof toggle.toggleStateZ>, "triggered" | "enabled"> {
  toggle: () => void;
}

export const use = ({ aetherKey, source, sink }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqualProps({ source, sink });
  const [, { triggered, enabled }, setState] = Aether.use({
    aetherKey,
    type: toggle.Toggle.TYPE,
    schema: toggle.toggleStateZ,
    initialState: {
      triggered: false,
      enabled: false,
      ...memoProps,
    },
  });

  useEffect(() => {
    setState((state) => ({ ...state, ...memoProps }));
  }, [memoProps, setState]);

  const handleToggle = useCallback(
    () =>
      setState((state) => ({
        ...state,
        triggered: !state.triggered,
      })),
    [setState],
  );

  return {
    toggle: handleToggle,
    triggered,
    enabled,
  };
};
