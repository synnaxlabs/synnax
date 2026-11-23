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
import type { color, text } from "@synnaxlabs/x";
import type { ReactElement, ReactNode, Ref } from "react";

import { CSS } from "@/css";

const HEIGHT_RATIO = 0.72;

/** @returns the height of a readout under a field of the given size. */
export const readoutHeight = (size: Component.Size): number =>
  Math.round(Component.HEIGHTS[size] * HEIGHT_RATIO);

/**
 * The text level of a readout under a field of each size: the largest level whose line
 * height fits the readout.
 */
export const READOUT_LEVELS: Record<Component.Size, text.Level> = {
  tiny: "tiny",
  small: "small",
  medium: "p",
  large: "h4",
  huge: "h3",
};

export interface ReadoutProps {
  size: Component.Size;
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
  value,
  units,
  color,
  valueRef,
}: ReadoutProps): ReactElement => {
  const level = READOUT_LEVELS[size];
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
