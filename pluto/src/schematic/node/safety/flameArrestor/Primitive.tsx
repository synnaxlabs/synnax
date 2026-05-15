// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@synnaxlabs/lyra/css";
import { type ReactElement } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";

export interface Props extends Base.DivProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 33, height: 69 };

export const Primitive = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Base.Div {...rest} className={CSS(CSS.B("flame-arrestor"), className)}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={7.575}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={92.425}
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
      <Base.Path
        d="M16.5 2.5L16.5 66.5M2.5 34.5H30.5M2.5 18.9848H30.5M2.5 50.0152H30.5M5.3 66.5H27.7C29.2464 66.5 30.5 65.1976 30.5 63.5909V5.40909C30.5 3.80244 29.2464 2.5 27.7 2.5H5.3C3.7536 2.5 2.5 3.80245 2.5 5.40909V63.5909C2.5 65.1976 3.7536 66.5 5.3 66.5Z"
        strokeLinecap="round"
      />
    </Base.SVG>
  </Base.Div>
);
