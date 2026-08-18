// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Label } from "@/schematic/node/flowmeters/Label";
import { type Props } from "@/schematic/node/flowmeters/props";

const DIMENSIONS = { width: 71, height: 36 };
const LABELS = { x: 58, y: 29 };

export const Electromagnetic = ({
  id,
  className,
  orientation = "right",
  color,
  scale = 1,
  ...rest
}: Props) => (
  <Primitive.Div
    {...rest}
    className={CSS.cx(CSS.B("flowmeter-Electromagnetic"), className)}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={4}
      top={6}
      right={98}
      bottom={91}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2" y="2" width="67" height="31" rx="2" />
      <Primitive.Path d="M47.5 17.5H55.5" strokeLinecap="round" />
      <Primitive.Path d="M15.5 17.5H23.5" strokeLinecap="round" />
      <Primitive.Path
        d="M23.5 17.5C23.5 16.9747 23.6035 16.4546 23.8045 15.9693C24.0055 15.484 24.3001 15.043 24.6716 14.6716C25.043 14.3001 25.484 14.0055 25.9693 13.8045C26.4546 13.6035 26.9747 13.5 27.5 13.5C28.0253 13.5 28.5454 13.6035 29.0307 13.8045C29.516 14.0055 29.957 14.3001 30.3284 14.6716C30.6999 15.043 30.9945 15.484 31.1955 15.9693C31.3965 16.4546 31.5 16.9747 31.5 17.5"
        strokeWidth="2"
      />
      <Primitive.Path
        d="M31.5 17.5C31.5 16.9747 31.6035 16.4546 31.8045 15.9693C32.0055 15.484 32.3001 15.043 32.6716 14.6716C33.043 14.3001 33.484 14.0055 33.9693 13.8045C34.4546 13.6035 34.9747 13.5 35.5 13.5C36.0253 13.5 36.5454 13.6035 37.0307 13.8045C37.516 14.0055 37.957 14.3001 38.3284 14.6716C38.6999 15.043 38.9945 15.484 39.1955 15.9693C39.3965 16.4546 39.5 16.9747 39.5 17.5"
        strokeWidth="2"
      />
      <Primitive.Path
        d="M39.5 17.5C39.5 16.9747 39.6035 16.4546 39.8045 15.9693C40.0055 15.484 40.3001 15.043 40.6716 14.6716C41.043 14.3001 41.484 14.0055 41.9693 13.8045C42.4546 13.6035 42.9747 13.5 43.5 13.5C44.0253 13.5 44.5454 13.6035 45.0307 13.8045C45.516 14.0055 45.957 14.3001 46.3284 14.6716C46.6999 15.043 46.9945 15.484 47.1955 15.9693C47.3965 16.4546 47.5 16.9747 47.5 17.5"
        strokeWidth="2"
      />
      <Label position={LABELS} color={color} />
    </Primitive.SVG>
  </Primitive.Div>
);
