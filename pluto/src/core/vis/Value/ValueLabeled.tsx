// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import { Box, XYScale, XY } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { DirectionTrigger, useResize } from "@/core/hooks";
import { PackProps, Space, Text } from "@/core/std";
import { Theming } from "@/core/theming";
import { ValueCore, ValueCoreProps } from "@/core/vis/Value/ValueCore";

import "@/core/vis/Value/ValueLabeled.css";

export interface ValueLabeledProps
  extends Omit<ValueCoreProps, "box">,
    Omit<PackProps, "color" | "onChange"> {
  position?: XY;
  label: string;
  onLabelChange?: (label: string) => void;
}

export const ValueLabeled = ({
  label,
  onLabelChange,
  level = "p",
  color,
  position,
  className,
  children,
  ...props
}: ValueLabeledProps): ReactElement => {
  const [box, setBox] = useState<Box>(Box.ZERO);

  const font = Theming.useTypography(level);

  const triggers: DirectionTrigger[] = position != null ? [] : ["resizeX", "resizeY"];
  const resizeRef = useResize(setBox, { triggers });

  const height = (font.lineHeight + 2) * font.baseSize;

  let adjustedBox = box;
  if (position != null)
    adjustedBox = XYScale.translate(position)
      .translate(box.topLeft.scale(-1))
      .translateY(1.9 * font.baseSize)
      .translateY(height)
      .box(box);

  return (
    <Space
      {...props}
      direction="y"
      className={CSS(className, CSS.BE("value-labeled", "container"))}
    >
      <Text.MaybeEditable
        value={label}
        onChange={onLabelChange}
        level={level}
        style={{
          textAlign: "center",
          padding: "1rem",
          width: "fit-content",
          minWidth: "100%",
        }}
      />
      <div className={CSS.B("value")} style={{ height, width: "100%" }} ref={resizeRef}>
        {children}
        <ValueCore color={color} level={level} {...props} box={adjustedBox} />
      </div>
    </Space>
  );
};
