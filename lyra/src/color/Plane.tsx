// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type color, type xy } from "@synnaxlabs/x";
import { type KeyboardEvent, type ReactElement, useRef } from "react";

import { keyStep, useNormalizedDrag } from "@/color/drag";
import { THUMB_VAR } from "@/color/Slider";
import { CSS } from "@/css";

const PERCENT = bounds.construct(0, 100);
const HUE_VAR = CSS.variable("color-picker", "hue");

export interface PlaneProps {
  /** The plane shows the hue and puts the thumb at the saturation and value. */
  value: color.HSVA;
  /** Called with the new saturation and value, each from 0 to 100. */
  onChange: (saturation: number, value: number) => void;
  /** The CSS color that fills the thumb. */
  thumb: string;
}

/**
 * The saturation and value square of an HSV picker: saturation runs left to right and
 * value runs bottom to top. Arrow keys move the thumb one percent, or ten with shift.
 */
export const Plane = ({ value, onChange, thumb }: PlaneProps): ReactElement => {
  const [hue, saturation, brightness] = value;
  const ref = useRef<HTMLDivElement>(null);
  useNormalizedDrag(ref, (p: xy.XY) => onChange(p.x * 100, (1 - p.y) * 100));
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>): void => {
    const step = keyStep(e);
    if (step == null) return;
    e.preventDefault();
    const horizontal = e.key === "ArrowLeft" || e.key === "ArrowRight";
    onChange(
      bounds.clamp(PERCENT, saturation + (horizontal ? step * 100 : 0)),
      bounds.clamp(PERCENT, brightness + (horizontal ? 0 : step * 100)),
    );
  };
  return (
    <div
      ref={ref}
      className={CSS.BE("color-picker", "plane")}
      style={{ [HUE_VAR]: hue }}
      role="slider"
      tabIndex={0}
      aria-label="Saturation and brightness"
      aria-valuetext={`Saturation ${Math.round(saturation)}%, brightness ${Math.round(
        brightness,
      )}%`}
      onKeyDown={handleKeyDown}
    >
      <div
        className={CSS.BE("color-picker", "thumb")}
        style={{
          left: `${saturation}%`,
          top: `${100 - brightness}%`,
          [THUMB_VAR]: thumb,
        }}
      />
    </div>
  );
};
