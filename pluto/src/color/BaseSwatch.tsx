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
import { type ReactElement, useCallback, useMemo } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { Theming } from "@/theming";

export const HAUL_TYPE = "color";

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
  "onChange" | "value" | "size"
> {
  value: color.Crude;
  onChange?: (c: color.Color) => void;
  size?: Button.ButtonProps["size"] | "tiny";
}

export const BaseSwatch = ({
  value,
  onChange,
  className,
  size = "medium",
  draggable = true,
  style,
  ...rest
}: BaseSwatchProps): ReactElement => {
  const background = Theming.use().colors.gray.l0;
  const clr = color.construct(value);
  const dragging = Haul.useDraggingState();
  const canDrop: Haul.CanDrop = useCallback(
    ({ items }) => {
      const [k] = filterHaulItems(items);
      return k != null && k.key !== color.hex(clr);
    },
    [clr],
  );
  const handleDrop: Haul.OnDrop = useCallback(
    ({ items }) => {
      const [k] = filterHaulItems(items);
      if (k != null) onChange?.(color.construct(k.key));
      return items;
    },
    [onChange],
  );
  const { startDrag, ...haulProps } = Haul.useDragAndDrop({
    type: "color_swatch",
    onDrop: handleDrop,
    canDrop,
  });
  const handleDragStart = useCallback(() => {
    startDrag([createHaulItem(color.hex(clr))]);
  }, [startDrag, clr]);
  const swatchStyle = useMemo(
    () => ({ ...style, [CSS.var("swatch", "color")]: color.cssString(value) }),
    [style, value],
  );
  return (
    <Button.Button
      className={CSS(
        CSS.B("color-swatch"),
        CSS.M(size),
        color.contrast(background, clr) > 1.5 &&
          color.aValue(clr) > 0.5 &&
          CSS.M("no-border"),
        CSS.dropRegion(canDrop(dragging)),
        className,
      )}
      size={size}
      draggable={draggable}
      onDragStart={handleDragStart}
      style={swatchStyle}
      variant="outlined"
      {...haulProps}
      {...rest}
    />
  );
};
