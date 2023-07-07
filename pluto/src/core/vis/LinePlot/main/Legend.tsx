// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useRef } from "react";

import { XY, CrudeXY, Box } from "@synnaxlabs/x";

import { useLinePlotContext } from "./LinePlot";

import { Color } from "@/core/color";
import { CSS } from "@/core/css";
import { useCursorDrag } from "@/core/hooks/useCursorDrag";
import { Input, Space, SpaceProps, Text } from "@/core/std";
import { PartialInputControl } from "@/core/std/Input/types";
import { preventDefault } from "@/util/event";

import "@/core/vis/LinePlot/main/Legend.css";

export interface LegendProps
  extends Omit<SpaceProps, "onChange">,
    Partial<PartialInputControl<CrudeXY>> {}

export const Legend = ({
  className,
  value,
  onChange,
  style,
  ...props
}: LegendProps): ReactElement | null => {
  const { lines } = useLinePlotContext("Legend");
  const [position, setPosition] = Input.usePassthrough({
    value,
    onChange,
    initialValue: new XY(50, 50).crude,
  });
  useLinePlotContext("LegendPosition");
  const positionRef = useRef(position);
  if (position !== null) {
    style = {
      ...style,
      ...new XY(position).css,
    };
  }

  const dragProps = useCursorDrag({
    onMove: useCallback(
      (box: Box) => {
        setPosition(new XY(positionRef.current).translate(box.signedDims));
      },
      [setPosition]
    ),
    onEnd: useCallback((box: Box) => {
      positionRef.current = new XY(positionRef.current).translate(box.signedDims);
    }, []),
  });

  if (lines.length === 0) return null;

  return (
    <Space
      className={CSS(className, CSS.B("legend"))}
      bordered
      rounded
      style={style}
      onDragStart={dragProps}
      draggable
      {...props}
      onDrag={preventDefault}
      onDragEnd={preventDefault}
      size="small"
    >
      {lines.map(({ key, color, label }) => (
        <Space key={key} direction="x" align="center">
          <div
            className={CSS(CSS.B("legend__color"), CSS.rounded())}
            style={{ backgroundColor: new Color(color).hex }}
          />
          <Text level="small">{label}</Text>
        </Space>
      ))}
    </Space>
  );
};
