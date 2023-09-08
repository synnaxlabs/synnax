// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { Box, Direction, type XY } from "@synnaxlabs/x";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { useResize } from "@/hooks";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { type UseTypographyReturn } from "@/theming/font";
import { Core, type CoreProps } from "@/vis/value/Core";

import "@/vis/value/Labeled.css";

export interface ValueLabeledProps
  extends Omit<CoreProps, "box">,
    Omit<Align.SpaceProps, "color" | "onChange"> {
  position?: XY;
  zoom?: number;
  label: string;
  onLabelChange?: (label: string) => void;
  color?: Color.Crude;
  textColor?: Color.Crude;
}

export const ValueLabeled = ({
  label,
  onLabelChange,
  level = "p",
  direction = "y",
  position,
  className,
  children,
  textColor,
  color,
  zoom = 1,
  ...props
}: ValueLabeledProps): ReactElement => {
  const font = Theming.useTypography(level);
  const [box, setBox] = useState<Box>(Box.ZERO);

  const valueBoxHeight = (font.lineHeight + 2) * font.baseSize + 2;
  const resizeRef = useResize(setBox, {});

  const adjustedBox = adjustBox(
    new Direction(direction),
    zoom,
    box,
    valueBoxHeight,
    font,
    position,
  );

  return (
    <Align.Space
      className={CSS(className, CSS.B("value-labeled"))}
      align="center"
      ref={resizeRef}
      direction={direction}
      {...props}
    >
      <Text.MaybeEditable value={label} onChange={onLabelChange} level={level} />
      <div
        className={CSS.B("value")}
        style={{
          height: valueBoxHeight,
          borderColor: Color.cssString(color),
        }}
      >
        {children}
        <Core color={textColor} level={level} {...props} box={adjustedBox} />
      </div>
    </Align.Space>
  );
};

const adjustBox = (
  direction: Direction,
  zoom: number,
  box: Box,
  valueBoxHeight: number,
  font: UseTypographyReturn,
  position?: XY,
): Box => {
  if (direction.isX) {
    return new Box(
      (position?.x ?? box.left) + box.width / zoom - 100,
      position?.y ?? box.top,
      100,
      valueBoxHeight,
    );
  }
  return new Box(
    position?.x ?? box.left,
    (position?.y ?? box.top) + box.height / zoom - valueBoxHeight,
    box.width / zoom,
    valueBoxHeight,
  );
};
