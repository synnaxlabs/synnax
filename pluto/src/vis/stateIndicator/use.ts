// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useMemoDeepEqual } from "@/memo";
import { stateIndicator } from "@/vis/stateIndicator/aether";

export interface UseProps extends Pick<
  z.input<typeof stateIndicator.stateZ>,
  "source"
> {
  aetherKey: string;
}

export interface UseReturn extends Pick<
  z.infer<typeof stateIndicator.stateZ>,
  "value"
> {}

export const use = ({ aetherKey, source }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ source });
  const [, { value }, setState] = Aether.use({
    aetherKey,
    type: stateIndicator.StateIndicator.TYPE,
    schema: stateIndicator.stateZ,
    initialState: { value: 0, ...memoProps },
  });
  useEffect(
    () => setState((state) => ({ ...state, ...memoProps })),
    [memoProps, setState],
  );
  return { value };
};
