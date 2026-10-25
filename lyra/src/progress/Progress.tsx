// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/progress/Progress.css";

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { type ComponentPropsWithRef, type ReactElement } from "react";

import { CSS } from "@/css";

interface ProgressProps extends Omit<ComponentPropsWithRef<"div">, "children"> {
  /** How far along the work is, from 0 to 100. Leave it out while that is unknown. */
  value?: number;
  /**
   * How long the work should take. Without a value, the bar fills over this span, then
   * pulses until it unmounts.
   */
  duration?: CrudeTimeSpan;
  /** Draws a 2px line with no track, to run along the edge of a bar or a pane. */
  thin?: boolean;
}

const DURATION_VARIABLE = CSS.variable("progress-duration");

/** A bar that shows how far along some work is. */
export const Progress = ({
  value,
  duration,
  thin = false,
  className,
  style,
  ...rest
}: ProgressProps): ReactElement => {
  const ms = duration == null ? 0 : new TimeSpan(duration).milliseconds;
  return (
    <div
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={value}
      className={CSS.cls(
        CSS.B("progress"),
        thin && CSS.M("thin"),
        value == null && CSS.M("indeterminate"),
        className,
      )}
      style={{ [DURATION_VARIABLE]: `${ms}ms`, ...style }}
      {...rest}
    >
      <div
        className={CSS.B("progress-bar")}
        style={value == null ? undefined : { width: `${value}%` }}
      />
    </div>
  );
};
