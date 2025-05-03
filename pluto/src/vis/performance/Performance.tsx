// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Aether } from "@/aether";
import { Text } from "@/text";
import { performance } from "@/vis/performance/aether";

export const Tools = () => {
  const [, state] = Aether.use({
    type: performance.Performance.TYPE,
    schema: performance.performanceStateZ,
    initialState: {
      fps: 0,
    },
  });

  return (
    <Text.Text shade={9} level="p" code>
      {state.fps} FPS
    </Text.Text>
  );
};
