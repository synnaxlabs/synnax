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
  "source" | "options" | "stalenessTimeout"
> {
  aetherKey: string;
}

export interface UseReturn extends Pick<
  z.infer<typeof stateIndicator.stateZ>,
  "key" | "stale"
> {}

export const use = ({
  aetherKey,
  source,
  options,
  stalenessTimeout,
}: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ source, options, stalenessTimeout });
  const [, { key, stale }, setState] = Aether.use({
    aetherKey,
    type: stateIndicator.StateIndicator.TYPE,
    schema: stateIndicator.stateZ,
    initialState: { key: null, ...memoProps },
  });
  useEffect(
    () => setState((state) => ({ ...state, ...memoProps })),
    [memoProps, setState],
  );
  return { key, stale };
};
