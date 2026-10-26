// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { stringValue } from "@/vis/stringValue/aether";

export interface UseProps extends z.input<typeof stringValue.stateZ> {
  aetherKey: string;
}

/**
 * use draws the latest value of a string source into a box on the canvas. The worker
 * draws each sample, so a sample never re-renders the calling component.
 */
export const use = ({ aetherKey, ...rest }: UseProps): void => {
  const memoProps = useMemoDeepEqual(rest);
  const { setState } = Aether.useLifecycle({
    aetherKey,
    type: stringValue.StringValue.TYPE,
    schema: stringValue.stateZ,
    initialState: memoProps,
  });
  useEffect(
    () => setState((state) => ({ ...state, ...memoProps })),
    [memoProps, setState],
  );
};
