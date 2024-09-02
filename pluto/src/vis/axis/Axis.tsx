// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scale } from "@synnaxlabs/x";
import { memo, useMemo } from "react";

import { axis } from "@/vis/axis/aether";

export interface SVGProps extends axis.AxisState {
  size: number;
  decimalToDataScale: scale.Scale;
}

export const SVG = memo(
  ({ tickSpacing, size, decimalToDataScale, type, ...props }: SVGProps) => {
    const tf = useMemo(
      () => axis.newTickFactory({ tickSpacing, type }),
      [tickSpacing, type],
    );

    const ticks = tf.generate({ size, decimalToDataScale });

    return <svg></svg>;
  },
);
SVG.displayName = "Axis.SVG";
