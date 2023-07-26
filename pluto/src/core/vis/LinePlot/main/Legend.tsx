// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, memo, useCallback, useRef, useState } from "react";

import { XY, CrudeXY, Box } from "@synnaxlabs/x";

import { useLinePlotContext } from "./LinePlot";

import { Color } from "@/core/color";
import { ColorSwatch } from "@/core/color/ColorSwatch";
import { CSS } from "@/core/css";
import { useCursorDrag } from "@/core/hooks/useCursorDrag";
import { Input, Space, SpaceProps, Text } from "@/core/std";
import { PartialInputControl } from "@/core/std/Input/types";
import { preventDefault } from "@/util/event";

import "@/core/vis/LinePlot/main/Legend.css";

export interface LegendProps
  extends Omit<SpaceProps, "onChange">,
    Partial<PartialInputControl<CrudeXY>> {
  onLabelChange?: (id: string, label: string) => void;
  onColorChange?: (id: string, color: Color) => void;
}

export const Legend = memo(
  ({
    className,
    value,
    onChange,
    style,
    onLabelChange,
    onColorChange,
    ...props
  }: LegendProps): ReactElement | null => {
    const { lines } = useLinePlotContext("Legend");
    const [position, setPosition] = Input.usePassthrough({
      value,
      onChange,
      initialValue: new XY(50, 50).crude,
    });
    const [pickerVisible, setPickerVisible] = useState(false);
    useLinePlotContext("LegendPosition");
    const positionRef = useRef(position);
    if (position !== null) {
      style = {
        ...style,
        ...new XY(position).css,
      };
    }

    const handleCursorDragStart = useCursorDrag({
      onMove: useCallback(
        (box: Box) => {
          if (!pickerVisible)
            setPosition(new XY(positionRef.current).translate(box.signedDims));
        },
        [setPosition, pickerVisible]
      ),
      onEnd: useCallback(
        (box: Box) => {
          if (!pickerVisible)
            positionRef.current = new XY(positionRef.current).translate(box.signedDims);
        },
        [pickerVisible]
      ),
    });

    if (lines.length === 0) return null;

    return (
      <Space
        className={CSS(className, CSS.B("legend"))}
        bordered
        rounded
        style={style}
        onDragStart={handleCursorDragStart}
        draggable
        {...props}
        onDrag={preventDefault}
        onDragEnd={preventDefault}
        size="small"
      >
        {lines.map(({ key, color, label }) => (
          <Space key={key} direction="x" align="center">
            <ColorSwatch
              value={color}
              onChange={(c) => onColorChange?.(key, c)}
              onVisibleChange={setPickerVisible}
              size="tiny"
            />
            <Text.MaybeEditable
              level="small"
              value={label}
              onChange={onLabelChange != null && ((l) => onLabelChange(key, l))}
              noWrap
            />
          </Space>
        ))}
      </Space>
    );
  }
);
Legend.displayName = "Legend";
