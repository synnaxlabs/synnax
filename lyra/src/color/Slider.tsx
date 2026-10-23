// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type xy } from "@synnaxlabs/x";
import { type KeyboardEvent, type ReactElement, useRef } from "react";

import { keyStep, useNormalizedDrag } from "@/color/drag";
import { CSS } from "@/css";

const UNIT = bounds.construct(0, 1);
const TRACK_VAR = CSS.variable("color-picker", "track");
export const THUMB_VAR = CSS.variable("color-picker", "thumb");

export interface SliderProps {
  /** The position of the thumb, from 0 to 1. */
  value: number;
  onChange: (value: number) => void;
  /** The CSS background of the track. */
  track: string;
  /** The CSS color that fills the thumb. */
  thumb: string;
  label: string;
  /** The value read out to assistive technology. */
  valueText: string;
  className?: string;
}

/** A horizontal color track with a round thumb, driven by pointer or arrow keys. */
export const Slider = ({
  value,
  onChange,
  track,
  thumb,
  label,
  valueText,
  className,
}: SliderProps): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  useNormalizedDrag(ref, (p: xy.XY) => onChange(p.x));
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>): void => {
    const step = keyStep(e);
    if (step == null) return;
    e.preventDefault();
    onChange(bounds.clamp(UNIT, value + step));
  };
  return (
    <div
      ref={ref}
      className={CSS.cls(CSS.BE("color-picker", "slider"), className)}
      style={{ [TRACK_VAR]: track }}
      role="slider"
      tabIndex={0}
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(value * 100)}
      aria-valuetext={valueText}
      onKeyDown={handleKeyDown}
    >
      <div
        className={CSS.BE("color-picker", "thumb")}
        style={{ left: `${value * 100}%`, [THUMB_VAR]: thumb }}
      />
    </div>
  );
};
