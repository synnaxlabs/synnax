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
import { button } from "@/vis/button/aether";

export interface UseProps extends Omit<z.input<typeof button.buttonStateZ>, "trigger"> {
  aetherKey: string;
}

export interface UseReturn {
  click: () => void;
}

export const use = ({ aetherKey, sink }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqualProps({ sink });
  const [, , setState] = Aether.use({
    aetherKey,
    type: button.Button.TYPE,
    schema: button.buttonStateZ,
    initialState: {
      trigger: 0,
      sink,
    },
  });

  useEffect(() => {
    setState((p) => ({ ...p, ...memoProps }));
  }, [memoProps]);

  const click = useCallback(() => {
    setState((p) => ({ ...p, trigger: p.trigger + 1 }));
  }, [setState]);

  return { click };
};
