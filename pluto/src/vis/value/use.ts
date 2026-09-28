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
import { Value } from "@/vis/value/aether/value";

export interface UseProps extends z.input<typeof Value.z> {
  aetherKey: string;
}

export const use = ({ aetherKey, ...props }: UseProps): void => {
  const memoProps = useMemoDeepEqual(props);
  const [, , setState] = Aether.use({
    aetherKey,
    type: Value.TYPE,
    schema: Value.z,
    initialState: memoProps,
  });
  useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);
};
