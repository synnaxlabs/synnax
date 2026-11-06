// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Text } from "@synnaxlabs/lyra/text";
import { type color, text } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, type Ref } from "react";

import { CSS } from "@/css";

const HEIGHT_RATIO = 0.72;

/** @returns the height of a readout under a field of the given size. */
export const readoutHeight = (size: Component.Size): number =>
  Math.round(Component.HEIGHTS[size] * HEIGHT_RATIO);

// The largest level whose line height fits a readout on each rung of the standard scale.
const READOUT_LEVELS: Record<Component.Size, text.Level> = {
  tiny: "tiny",
  small: "small",
  medium: "p",
  large: "h4",
  huge: "h3",
};

/**
 * @param size - The size of the field above the readout.
 * @param level - The text level of the field. Absent keeps Lyra's default for the size.
 * @returns the text level of a readout under the field.
 */
export const readoutLevel = (size: Component.Size, level?: text.Level): text.Level =>
  level == null
    ? text.downLevel(Component.SIZE_TEXT_LEVELS[size])
    : READOUT_LEVELS[size];

export interface ReadoutProps {
  size: Component.Size;
  /** The text level of the field above the readout. */
  level?: text.Level;
  value?: ReactNode;
  units?: string;
  color?: color.Color;
  /** valueRef receives the value cell, so a value drawn on the canvas can fit it. */
  valueRef?: Ref<HTMLParagraphElement>;
}

/**
 * Readout is a read-only tray that hangs under a field and shows the value a device
 * reports.
 */
export const Readout = ({
  size,
  level: fieldLevel,
  value,
  units,
  color,
  valueRef,
}: ReadoutProps): ReactElement => {
  const level = readoutLevel(size, fieldLevel);
  return (
    <div
      className={CSS.BE("symbol-field", "readout")}
      style={{ height: readoutHeight(size) }}
    >
      <Text.Text
        ref={valueRef}
        level={level}
        color={color}
        overflow="fade"
        className={CSS.BE("symbol-field", "readout-value")}
      >
        {value}
      </Text.Text>
      {units != null && units.length > 0 && (
        <Text.Text level={level} color={9}>
          {units}
        </Text.Text>
      )}
    </div>
  );
};
