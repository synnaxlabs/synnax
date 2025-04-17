// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { Text } from "@/text";
import { Legend } from "@/vis/legend";
import { type StickyXY } from "@/vis/legend/Container";
import { performance } from "@/vis/performance/aether";

const INITIAL_POSITION: StickyXY = {
  x: 30,
  y: 30,
  root: location.BOTTOM_RIGHT,
  units: { x: "px", y: "px" },
};

export const Performance = () => {
  const [, state] = Aether.use({
    type: performance.Performance.TYPE,
    schema: performance.performanceStateZ,
    initialState: {
      fps: 0,
    },
  });

  return (
    <Legend.Container
      background={2}
      initial={INITIAL_POSITION}
      style={{ zIndex: 100, width: 200, boxShadow: "var(--pluto-shadow-v1)" }}
    >
      <Text.Text shade={9} level="p" code>
        {state.fps} FPS
      </Text.Text>
    </Legend.Container>
  );
};
