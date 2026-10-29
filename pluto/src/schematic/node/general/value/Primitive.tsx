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
import { Text } from "@synnaxlabs/lyra/text";
import { color, type dimensions, type text } from "@synnaxlabs/x";
import { type CSSProperties, type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

interface RenderProps extends PropsWithChildren<
  Pick<
    schematic.ValueNodeConfig,
    "strokeColor" | "orientation" | "units" | "inlineSize"
  >
> {
  className?: string;
  dimensions?: dimensions.Dimensions;
  unitsLevel?: text.Level;
}

export const Value = ({
  className,
  strokeColor,
  dimensions,
  orientation,
  units,
  unitsLevel = "small",
  children,
  inlineSize = 80,
}: RenderProps): ReactElement => {
  const symbolColor = color.rgbaString(strokeColor);
  const style: CSSProperties = {
    [Primitive.SYMBOL_COLOR]: symbolColor,
    height: dimensions?.height,
  };
  const contentStyle: CSSProperties = {
    minWidth: dimensions?.width,
    inlineSize,
    maxWidth: dimensions?.width,
  };
  return (
    <Primitive.Div
      className={CSS.cls(
        CSS.B("value"),
        CSS.symbolColored,
        CSS.B("symbol-readout"),
        symbolColor != null && CSS.M("colored"),
        className,
      )}
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
      {units.length > 0 && (
        <div className={CSS.cls(CSS.BE("value", "units"), CSS.M(unitsLevel))}>
          <Text.Text level={unitsLevel} color={9}>
            {units}
          </Text.Text>
        </div>
      )}
    </Primitive.Div>
  );
};
