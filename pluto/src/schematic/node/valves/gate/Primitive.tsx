// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@synnaxlabs/charon";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
export interface Props extends Toggle.ButtonProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 87, height: 42 };

export const Primitive = ({
  color,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("gate-valve"), className)}
    enabled={enabled}
  >
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={50}
        id="2"
      />
    </Handle.Boundary>
    <Base.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Base.Path d="M43.5 21L6.35453 2.20349C4.35901 1.19372 2 2.64384 2 4.88029V37.1197C2 39.3562 4.35901 40.8063 6.35453 39.7965L43.5 21ZM43.5 21L80.6455 2.20349C82.641 1.19372 85 2.64384 85 4.8803V37.1197C85 39.3562 82.641 40.8063 80.6455 39.7965L43.5 21Z" />
      <Base.Path d="M43.5 2 L43.5 39.7965" />
    </Base.SVG>
  </Toggle.Button>
);
