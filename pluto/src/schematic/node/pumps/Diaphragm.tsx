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

export interface DiaphragmProps extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 64, height: 64 };

export const Diaphragm = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: DiaphragmProps): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cls(CSS.B("pump"), className)}
    orientation={orientation}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={3.125}
      top={3.125}
      right={96.875}
      bottom={96.875}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path
        d="M62 32C62 48.5685 48.5685 62 32 62M62 32C62 15.4315 48.5685 2 32 2C15.4315 2 2 15.4315 2 32C2 48.5685 15.4315 62 32 62M62 32L32 62M32 2.00269L62.0025 32.0052"
        strokeLinecap="round"
      />
      <Primitive.Path
        d="M31 62C29.2204 62 27.3855 61.332 25.5927 59.9086C23.791 58.4782 22.0952 56.3316 20.6377 53.5381C19.1829 50.7496 18.0147 47.4106 17.214 43.7054C16.4137 40.0021 16 36.0237 16 32C16 27.9763 16.4137 23.9979 17.214 20.2946C18.0147 16.5895 19.1829 13.2504 20.6377 10.4619C22.0952 7.66841 23.791 5.52179 25.5927 4.09136C27.3855 2.66801 29.2204 2 31 2"
        strokeWidth="2"
      />
    </Primitive.SVG>
  </Toggle.Button>
);
