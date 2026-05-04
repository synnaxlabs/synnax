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
import {
  Handle,
  HandleBoundary,
  InternalSVG,
  Line,
  Path,
  Rect,
  type SVGBasedPrimitiveProps,
  Toggle,
  type ToggleProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends ToggleProps, SVGBasedPrimitiveProps {
  normallyOpen?: boolean;
}

const DIMENSIONS = { width: 87, height: 69 };

export const Primitive = ({
  className,
  color,
  orientation = "left",
  normallyOpen = false,
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle
    className={CSS(
      CSS.B("solenoid-valve"),
      normallyOpen && CSS.M("normally-open"),
      className,
    )}
    orientation={orientation}
    {...rest}
  >
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={69.5652}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={69.5652}
        id="2"
      />
      <Handle
        location="left"
        orientation={orientation}
        left={33.3333}
        top={17.7778}
        id="3"
      />
      <Handle location="top" orientation={orientation} left={50} top={2.8986} id="4" />
      <Handle
        location="right"
        orientation={orientation}
        left={66.6667}
        top={17.7778}
        id="5"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path
        className={CSS.B("body")}
        d="M43.5 48L6.35453 29.2035C4.35901 28.1937 2 29.6438 2 31.8803V64.1197C2 66.3562 4.35901 67.8063 6.35453 66.7965L43.5 48ZM43.5 48L80.6455 29.2035C82.641 28.1937 85 29.6438 85 31.8803V64.1197C85 66.3562 82.641 67.8063 80.6455 66.7965L43.5 48Z"
      />
      <Line x1={43.5} x2={43.5} y1={24.5333} y2={48} />
      <Rect x="29" y="2" width="29" height="22.5333" rx="1" />
    </InternalSVG>
  </Toggle>
);
