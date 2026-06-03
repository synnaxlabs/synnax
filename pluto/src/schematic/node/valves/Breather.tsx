// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { useColor } from "@/schematic/node/common/color";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 99, height: 57 };

export const Breather = ({
  color: colorVal,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => {
  const resolved = useColor(colorVal);
  return (
    <Toggle.Button
      {...rest}
      orientation={orientation}
      className={CSS(CSS.B("breather-valve"), className)}
      enabled={enabled}
    >
      <Handle.Linear orientation={orientation} left={8.081} right={91.919} />
      <Primitive.SVG
        dimensions={DIMENSIONS}
        color={resolved}
        orientation={orientation}
        scale={scale}
      >
        <Primitive.Circle cx="91" cy="49.5" r="6" fill={color.cssString(resolved)} />
        <Primitive.Circle cx="8" cy="7.5" r="6" fill={color.cssString(resolved)} />
        <Primitive.Path d="M49.5 28.5L12.3545 9.70349C10.359 8.69372 8 10.1438 8 12.3803V44.6197C8 46.8562 10.359 48.3063 12.3545 47.2965L49.5 28.5ZM49.5 28.5L86.6455 9.70349C88.641 8.69372 91 10.1438 91 12.3803V44.6197C91 46.8562 88.641 48.3063 86.6455 47.2965L49.5 28.5Z" />
      </Primitive.SVG>
    </Toggle.Button>
  );
};
