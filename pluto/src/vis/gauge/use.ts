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
import { text } from "@/text/core";
import { gauge } from "@/vis/gauge/aether";

export const corePropsZ = gauge.Gauge.z
  .partial({ color: true })
  .extend({ level: text.levelZ.optional() });

export interface UseProps extends z.input<typeof corePropsZ> {
  aetherKey: string;
}

export const use = ({
  aetherKey,
  box,
  telem,
  color,
  precision,
  minWidth,
  level = "small",
  backgroundTelem,
  notation,
  location,
  units,
  bounds,
  barWidth,
}: UseProps): void => {
  const memoProps = useMemoDeepEqual({
    box,
    telem,
    color,
    precision,
    level,
    minWidth,
    notation,
    backgroundTelem,
    location,
    units,
    bounds,
    barWidth,
  });
  const [, , setState] = Aether.use({
    aetherKey,
    type: gauge.Gauge.TYPE,
    schema: gauge.Gauge.z,
    initialState: memoProps,
  });
  useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);
};
