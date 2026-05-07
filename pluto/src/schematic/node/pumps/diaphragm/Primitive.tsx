// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@synnaxlabs/charon/css";
import { type ReactElement } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
export interface Props extends Toggle.ButtonProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 64, height: 64 };

export const Primitive = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS(CSS.B("pump"), className)}
    orientation={orientation}
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
        top={96.875}
        id="4"
      />
    </Handle.Boundary>
    <Base.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Base.Path
        d="M62 32C62 48.5685 48.5685 62 32 62M62 32C62 15.4315 48.5685 2 32 2C15.4315 2 2 15.4315 2 32C2 48.5685 15.4315 62 32 62M62 32L32 62M32 2.00269L62.0025 32.0052"
        strokeLinecap="round"
      />
      <Base.Path
        d="M31 62C29.2204 62 27.3855 61.332 25.5927 59.9086C23.791 58.4782 22.0952 56.3316 20.6377 53.5381C19.1829 50.7496 18.0147 47.4106 17.214 43.7054C16.4137 40.0021 16 36.0237 16 32C16 27.9763 16.4137 23.9979 17.214 20.2946C18.0147 16.5895 19.1829 13.2504 20.6377 10.4619C22.0952 7.66841 23.791 5.52179 25.5927 4.09136C27.3855 2.66801 29.2204 2 31 2"
        strokeWidth="2"
      />
    </Base.SVG>
  </Toggle.Button>
);
