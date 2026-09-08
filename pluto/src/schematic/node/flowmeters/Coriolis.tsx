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
import { Label } from "@/schematic/node/flowmeters/Label";
import { type Props } from "@/schematic/node/flowmeters/props";

const DIMENSIONS = { width: 71, height: 36 };
const LABELS = { x: 58, y: 29 };

export const Coriolis = ({
  id,
  className,
  orientation = "right",
  color,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("flowmeter-Coriolis"), className)}>
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
      <Primitive.Path d="M2 17.6024H28.5" strokeLinecap="round" />
      <Primitive.Path d="M28.5 17.6024L34.6834 14.0324" strokeLinecap="round" />
      <Primitive.Path d="M34.8 14L45.9058 20.9666" strokeLinecap="round" />
      <Primitive.Path d="M51.5 17.6024L46.0141 20.8987" strokeLinecap="round" />
      <Primitive.Path d="M20.5 17.6024L26.6574 14.0474" strokeLinecap="round" />
      <Primitive.Path d="M26.75 14.1024L37.788 21.0265" strokeLinecap="round" />
      <Primitive.Path d="M43.5 17.6024L37.8427 21.0017" strokeLinecap="round" />
      <Primitive.Path d="M43.5 17.6024H69" strokeLinecap="round" />
      <Label position={LABELS} color={color} />
    </Primitive.SVG>
  </Primitive.Div>
);
