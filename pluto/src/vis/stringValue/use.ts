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
import { stringValue } from "@/vis/stringValue/aether";

export interface UseProps
  extends Pick<z.input<typeof stringValue.stateZ>, "telem" | "stalenessTimeout"> {
  aetherKey: string;
}

export interface UseReturn
  extends Pick<z.infer<typeof stringValue.stateZ>, "value" | "stale"> {}

export const use = ({ aetherKey, telem, stalenessTimeout }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ telem, stalenessTimeout });
  const [, { value, stale }, setState] = Aether.use({
    aetherKey,
    type: stringValue.StringValue.TYPE,
    schema: stringValue.stateZ,
    initialState: { value: "", stale: false, ...memoProps },
  });
  useEffect(
    () => setState((state) => ({ ...state, ...memoProps })),
    [memoProps, setState],
  );
  return { value, stale };
};
