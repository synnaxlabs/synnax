// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { LinePlot } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";

type Mode = lineplot.TimeMode;

const ICON_CLASS = CSS.BE("line-plot", "toolbar", "time-mode-icon");

const ICONS: Record<Mode, ReactElement> = {
  absolute: <Icon.Time className={ICON_CLASS} />,
  relative: <Icon.Timeline className={ICON_CLASS} />,
};

const LABELS: Record<Mode, string> = { absolute: "Absolute", relative: "Relative" };

const TOOLTIPS: Record<Mode, string> = {
  absolute: "Absolute time",
  relative: "Relative time: each range from its T0",
};

/**
 * Selects the time mode of the plot's x axis: absolute or relative. Renders nothing
 * when the axis does not show time. With the compare flag off, it renders only on a
 * relative axis, so the user can go back to absolute.
 */
export const TimeModeSelect = (): ReactElement | null => {
  const key = LinePlot.useKey();
  const { dispatch } = LinePlot.useDispatch();
  const { type, timeMode } = LinePlot.useXAxis({ axisKey: "x1" });
  const handleChange = (next: Mode) =>
    dispatch({
      key,
      actions: [
        lineplot.setAxisTimeMode({
          key: "x1",
          timeMode: next === "absolute" ? undefined : next,
        }),
      ],
    });
  const item = (mode: Mode) => (
    <Select.Item itemKey={mode} size="medium" tooltip={TOOLTIPS[mode]}>
      {ICONS[mode]}
      <span className={CSS.BE("line-plot", "toolbar", "time-mode-label")}>
        {LABELS[mode]}
      </span>
    </Select.Item>
  );
  if (type !== "time" || (!FLAGS.compare && timeMode !== "relative")) return null;
  return (
    <Select.Buttons<Mode>
      variant="outlined"
      value={timeMode ?? "absolute"}
      onChange={handleChange}
      className={CSS.BE("line-plot", "toolbar", "time-mode")}
    >
      {item("absolute")}
      {item("relative")}
    </Select.Buttons>
  );
};
