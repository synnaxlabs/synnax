// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Swatch.css";

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { Theming } from "@/theming";

export const HAUL_TYPE = "color";

const COLOR_VAR = CSS.variable("swatch", "color");
const TEXT_VAR = CSS.variable("swatch", "text");

export type HaulItem = Haul.Item<typeof HAUL_TYPE, color.Hex, undefined>;

export const createHaulItem = (key: color.Hex): HaulItem => ({
  type: HAUL_TYPE,
  key,
});

export const isHaulItem = (item: Haul.Item): item is HaulItem =>
  item.type === HAUL_TYPE;

export const filterHaulItems = (items: Haul.Item[]): HaulItem[] =>
  items.filter(isHaulItem);

export const canDropHaulItem = Haul.canDropOfType<HaulItem>(HAUL_TYPE);

export interface BaseSwatchProps extends Omit<
  Button.ButtonProps,
  "onChange" | "value"
> {
  value: color.Crude;
  /** Called with a dropped color. Without it, the swatch takes no drop. */
  onChange?: (c: color.Color) => void;
}

export const BaseSwatch = ({
  value,
  onChange,
  className,
  draggable = true,
  style,
  ...rest
}: BaseSwatchProps): ReactElement => {
  const { gray } = Theming.use().colors;
  const hex = color.hex(value);
  const dragging = Haul.useDraggingState();
  const canDrop: Haul.CanDrop = ({ items }) => {
    const [k] = filterHaulItems(items);
    return onChange != null && k != null && k.key !== hex;
  };
  const { startDrag, ...haulProps } = Haul.useDragAndDrop({
    type: "color_swatch",
    onDrop: ({ items }) => {
      const [k] = filterHaulItems(items);
      if (k != null) onChange?.(color.construct(k.key));
      return items;
    },
    canDrop,
  });
  return (
    <Button.Button
      className={CSS.cls(
        CSS.B("color-swatch"),
        CSS.dropRegion(canDrop(dragging)),
        className,
      )}
      square
      draggable={draggable}
      onDragStart={() => startDrag([createHaulItem(hex)])}
      style={{
        ...style,
        [COLOR_VAR]: color.cssString(value),
        [TEXT_VAR]: color.cssString(color.pickByContrast(value, gray.l0, gray.l11)),
      }}
      variant="outlined"
      {...haulProps}
      {...rest}
    />
  );
};
