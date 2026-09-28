// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/value/value.css";

import { type schematic } from "@synnaxlabs/client";
import { CSS } from "@synnaxlabs/lyra/css";
import { Text } from "@synnaxlabs/lyra/text";
import { color, type text } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type PropsWithChildren,
  type ReactElement,
  useMemo,
} from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

/**
 * Width of the border the symbol draws around the value. The canvas box the value
 * renders into sits inside of it, so the two must agree.
 */
export const BORDER_WIDTH = 2;

interface RenderProps extends PropsWithChildren<
  Pick<schematic.ValueNodeConfig, "color" | "orientation" | "units" | "inlineSize">
> {
  height?: number;
  unitsLevel?: text.Level;
}

export const Value = ({
  color: colorVal,
  height,
  orientation,
  units,
  unitsLevel = "small",
  children,
  inlineSize,
}: RenderProps): ReactElement => {
  const symbolColor = color.rgbaString(colorVal);
  const style = useMemo<CSSProperties>(
    () => ({
      [CSS.variable("symbol-color")]: symbolColor,
      [CSS.variable("value-border-width")]: `${BORDER_WIDTH}px`,
      height,
    }),
    [symbolColor, height],
  );
  const contentStyle = useMemo<CSSProperties>(() => ({ inlineSize }), [inlineSize]);
  return (
    <Primitive.Div
      className={CSS.cls(CSS.B("value"), CSS.B("symbol-colored"))}
      style={style}
    >
      <div className={CSS.BE("value", "content")} style={contentStyle}>
        {children}
      </div>
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={-2}
        right={100}
        bottom={102}
      />
      <div className={CSS.cls(CSS.BE("value", "units"), CSS.M(unitsLevel))}>
        <Text.Text level={unitsLevel}>{units}</Text.Text>
      </div>
    </Primitive.Div>
  );
};
