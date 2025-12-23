// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Aether } from "@/aether";
import { input } from "@/vis/input/aether";

export interface UseProps extends Pick<input.State, "sink"> {
  aetherKey: string;
}

export interface UseReturn {
  set: (value: string) => void;
}

export const use = ({ aetherKey, sink }: UseProps): UseReturn => {
  const { methods } = Aether.useUnidirectional({
    aetherKey,
    type: input.Input.TYPE,
    schema: input.stateZ,
    state: { sink },
    methods: input.methodsZ,
  });

  return { set: methods.set };
};
