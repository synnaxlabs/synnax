// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/icon/Meter.css";

import { CSS } from "@/css";
import { type FC, wrapSVGIcon } from "@/icon/Icon";

const BAR_HEIGHTS = [6, 10, 14];
const EMPTY_CLASS = CSS.BE("icon", "meter-empty");

/* #__NO_SIDE_EFFECTS__ */
const createMeter = (name: string, filled: number): FC =>
  wrapSVGIcon(
    (props) => (
      <svg
        viewBox="0 0 16 16"
        xmlns="http://www.w3.org/2000/svg"
        stroke="currentColor"
        fill="currentColor"
        strokeWidth={0}
        {...props}
      >
        {BAR_HEIGHTS.map((height, i) => (
          <rect
            key={height}
            className={i < filled ? undefined : EMPTY_CLASS}
            x={1.5 + i * 5}
            y={15 - height}
            width={3}
            height={height}
            rx={1}
          />
        ))}
      </svg>
    ),
    name,
  );

/**
 * A level as three rising bars, filled up to the level. The filled bars take the icon's
 * color, and the empty bars stay gray.
 */
export const Meter = {
  Low: createMeter("meter-low", 1),
  Medium: createMeter("meter-medium", 2),
  High: createMeter("meter-high", 3),
};
