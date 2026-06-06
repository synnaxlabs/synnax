// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 87, height: 42 };

export const Needle = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("needle-valve"), className)}
    enabled={enabled}
  >
    <Handle.Linear
      orientation={orientation}
      left={2.2989}
      right={97.7011}
      top={51.1905}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={colorVal}
      scale={scale}
    >
      <Primitive.Path
        d="M43.0152 21.5391L38.237 2.62245C38.1573 2.30658 38.396 2 38.7218 2L48.2782 2C48.604 2 48.8427 2.30658 48.763 2.62245L43.9848 21.5391C43.8576 22.0425 43.1424 22.0425 43.0152 21.5391Z"
        fill="var(--pluto-symbol-display)"
      />
      <Primitive.Path d="M43.5 21.5L6.35453 2.70349C4.35901 1.69372 2 3.14384 2 5.38029V37.6197C2 39.8562 4.35901 41.3063 6.35453 40.2965L43.5 21.5ZM43.5 21.5L80.6455 2.70349C82.641 1.69372 85 3.14384 85 5.3803V37.6197C85 39.8562 82.641 41.3063 80.6455 40.2965L43.5 21.5Z" />
    </Primitive.SVG>
  </Toggle.Button>
);
