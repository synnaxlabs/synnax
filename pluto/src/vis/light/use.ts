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
import { light } from "@/vis/light/aether";

export interface UseProps extends Pick<
  z.input<typeof light.stateZ>,
  "source" | "stalenessTimeout"
> {
  aetherKey: string;
}

export interface UseReturn extends Pick<
  z.infer<typeof light.stateZ>,
  "enabled" | "stale"
> {}

export const use = ({ aetherKey, source, stalenessTimeout }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ source, stalenessTimeout });
  const [, { enabled, stale }, setState] = Aether.use({
    aetherKey,
    type: light.Light.TYPE,
    schema: light.stateZ,
    initialState: { enabled: false, ...memoProps },
  });
  useEffect(
    () => setState((state) => ({ ...state, ...memoProps })),
    [memoProps, setState],
  );
  return { enabled, stale };
};
