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

export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, undefined>;

const createHaulItem = (key: string): HaulItem => ({
  type: HAUL_TYPE,
  key,
});

const FACTS = ["Color"];

const filterHaulItems = (items: Haul.Item[]): HaulItem[] =>
  items.filter((item): item is HaulItem => item.type === HAUL_TYPE);

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
  disabled,
  style,
  ...rest
}: BaseSwatchProps): ReactElement => {
  const { gray } = Theming.use().colors;
  const hex = color.hex(value);
  // A clear fill shows the surface, so its text takes the icon gray instead. The
  // l2 and l5 checkerboard shows through a translucent fill, so the icon takes the gray
  // that reads best over both of its squares.
  const opacity = color.opacity(value);
  const contrastOverCheckerboard = (icon: color.Color): number =>
    Math.min(
      color.contrast(color.over(value, gray.l2, opacity), icon),
      color.contrast(color.over(value, gray.l5, opacity), icon),
    );
  const text =
    opacity === 0
      ? gray.l9
      : contrastOverCheckerboard(gray.l0) >= contrastOverCheckerboard(gray.l11)
        ? gray.l0
        : gray.l11;
  const dragging = Haul.useDraggingState();
  const canDrop: Haul.CanDrop = ({ items }) => {
    const [k] = filterHaulItems(items);
    return disabled !== true && onChange != null && k != null && k.key !== hex;
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
      disabled={disabled}
      draggable={draggable}
      onDragStart={(event) =>
        startDrag({
          event,
          items: [createHaulItem(hex)],
          preview: (
            <Haul.Preview swatch={color.cssString(value)} name={hex} facts={FACTS} />
          ),
        })
      }
      style={{
        ...style,
        [COLOR_VAR]: color.cssString(value),
        [TEXT_VAR]: color.cssString(text),
      }}
      variant="outlined"
      {...haulProps}
      {...rest}
    />
  );
};
