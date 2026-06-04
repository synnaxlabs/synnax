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

const DIMENSIONS = { width: 108, height: 96 };

export const FourWay = ({
  className,
  orientation = "left",
  scale,
  color: colorVal,
  ...rest
}: Props): ReactElement => (
    <Toggle.Button
      {...rest}
      orientation={orientation}
      className={CSS(CSS.B("four-way-valve"), className)}
    >
      <Handle.Rectangle
        orientation={orientation}
        left={12.037}
        top={7.2916}
        right={87.963}
        bottom={92.6084}
      />
      <Primitive.SVG
        dimensions={DIMENSIONS}
        color={colorVal}
        scale={scale}
        orientation={orientation}
      >
        <Primitive.Path
          d="M3.02937 72.7038C2.50936 73.1041 2.50936 73.8883 3.02937 74.2886L7.14001 77.453C7.79757 77.9592 8.75 77.4904 8.75 76.6606V70.3318C8.75 69.502 7.79757 69.0332 7.14001 69.5394L3.02937 72.7038Z"
          fill="var(--pluto-symbol-display)"
        />
        <Primitive.Path d="M54.2 48L17.0545 29.2035C15.059 28.1937 12.7 29.6438 12.7 31.8803V64.1197C12.7 66.3562 15.059 67.8063 17.0545 66.7965L54.2 48ZM54.2 48L91.3455 29.2035C93.341 28.1937 95.7 29.6438 95.7 31.8803V64.1197C95.7 66.3562 93.341 67.8063 91.3455 66.7965L54.2 48Z" />
        <Primitive.Path d="M54.2 48L35.4035 85.1455C34.3937 87.141 35.8439 89.5 38.0803 89.5H70.3197C72.5562 89.5 74.0063 87.141 72.9965 85.1455L54.2 48ZM54.2 48L35.4035 10.8545C34.3937 8.85901 35.8439 6.5 38.0803 6.5H70.3197C72.5562 6.5 74.0063 8.85901 72.9965 10.8545L54.2 48Z" />
        <Primitive.Path
          d="M8.70001 73.5C24.7 73.5 28.7 86.8333 28.7 93.5"
          strokeLinecap="round"
        />
        <Primitive.Path
          d="M105.371 23.2962C105.891 22.8959 105.891 22.1117 105.371 21.7114L101.26 18.547C100.602 18.0408 99.65 18.5096 99.65 19.3394V25.6682C99.65 26.498 100.602 26.9668 101.26 26.4606L105.371 23.2962Z"
          fill="var(--pluto-symbol-display)"
        />
        <Primitive.Path
          d="M99.7 22.5C83.7 22.5 79.7 9.16667 79.7 2.5"
          strokeLinecap="round"
        />
      </Primitive.SVG>
    </Toggle.Button>
  );
