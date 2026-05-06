// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@synnaxlabs/charon";
import { type ReactElement } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/light/config";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
  enabled?: boolean;
}

const DIMENSIONS = { width: 64, height: 64 };

export const Primitive = ({
  className,
  color,
  orientation = "left",
  enabled,
  scale,
}: RenderProps): ReactElement => (
  <Base.Div
    orientation={orientation}
    className={CSS(CSS.B("light"), enabled && CSS.M("enabled"), className)}
  >
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={3.125}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50}
        top={3.125}
        id="3"
      />
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.75}
        id="4"
      />
    </Handle.Boundary>
    <Base.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Base.Circle cx="32" cy="32" r="30" />
    </Base.SVG>
  </Base.Div>
);
