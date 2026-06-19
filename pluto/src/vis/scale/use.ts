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
import { scale } from "@/vis/scale/aether";

export interface UseProps extends z.input<typeof scale.Scale.z> {
  aetherKey: string;
}

export const use = ({ aetherKey, ...props }: UseProps): void => {
  const memoProps = useMemoDeepEqual(props);
  const [, , setState] = Aether.use({
    aetherKey,
    type: scale.Scale.TYPE,
    schema: scale.Scale.z,
    initialState: memoProps,
  });
  useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);
};
