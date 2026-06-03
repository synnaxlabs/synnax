// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/value/value.css";

import { color, type dimensions, type text } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/css";
import { resolveColor } from "@/schematic/node/common/color";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/value/config";
import { Text } from "@/text";
import { Theming } from "@/theming";

interface RenderProps extends PropsWithChildren<Omit<Config, "label" | "variant">> {
  className?: string;
  dimensions?: dimensions.Dimensions;
  unitsLevel?: text.Level;
}

export const Value = ({
  className,
  color: colorVal,
  dimensions,
  orientation = "left",
  units = "psi",
  unitsLevel = "small",
  children,
  inlineSize = 80,
}: RenderProps): ReactElement => {
  const theme = Theming.use();
  const resolved = resolveColor(colorVal, theme);
  const borderColor = color.cssString(resolved);
  const textColor = color.cssString(
    color.pickByContrast(resolved, theme.colors.gray.l0, theme.colors.gray.l11),
  );
  return (
    <Primitive.Div
      className={CSS(CSS.B("value"), className)}
      style={{
        borderColor,
        height: dimensions?.height,
      }}
    >
      <div
        className={CSS.BE("value", "content")}
        style={{
          flexGrow: 1,
          minWidth: dimensions?.width,
          inlineSize,
          maxWidth: dimensions?.width,
        }}
      >
        {children}
      </div>
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={-2}
        right={100}
        bottom={102}
      />
      <div
        className={CSS(CSS.BE("value", "units"), CSS.M(unitsLevel))}
        style={{ background: borderColor }}
      >
        <Text.Text level={unitsLevel} color={textColor}>
          {units}
        </Text.Text>
      </div>
    </Primitive.Div>
  );
};
