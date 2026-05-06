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
import { Flowmeter } from "@/schematic/node/common/flowmeter";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
export interface Props extends Base.DivProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 71, height: 36 };

export const Primitive = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Base.Div {...rest} className={CSS(CSS.B("flowmeter-Pulse"), className)}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={4}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={98}
        top={50}
        id="2"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50}
        top={6}
        id="3"
      />
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={91}
        id="4"
      />
    </Handle.Boundary>
    <Base.SVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Base.Rect x="2" y="2" width="67" height="31" rx="2" />
      <Base.Path d="M31 13.5H39" strokeLinecap="round" />
      <Base.Path d="M23 21.5H31" strokeLinecap="round" />
      <Base.Path d="M39 21.5H47" strokeLinecap="round" />
      <Base.Path d="M39 13.5V21.5" strokeLinecap="round" />
      <Base.Path d="M31 13.5V21.5" strokeLinecap="round" />
      <Flowmeter.Label color={colorVal} />
    </Base.SVG>
  </Base.Div>
);
