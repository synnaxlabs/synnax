// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/editor/SelectPerformance.css";

import { type arc } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

type ManualPerformance = Exclude<arc.task.Performance, "auto">;

/** The number of meter bars each level fills. */
const FILLED: Record<ManualPerformance, number> = { low: 1, medium: 2, high: 3 };

const BAR_HEIGHTS = [6, 10, 14];
const BAR_CLASS = CSS.BE("performance-select", "bar");

/** Creates an icon that shows a level as three rising bars, filled up to the level. */
const createMeter = (level: ManualPerformance): Icon.FC =>
  Icon.wrapSVGIcon(
    (props) => (
      <svg viewBox="0 0 16 16" {...props}>
        {BAR_HEIGHTS.map((height, i) => (
          <rect
            key={height}
            className={CSS.cls(BAR_CLASS, i < FILLED[level] && CSS.M("filled"))}
            x={1.5 + i * 5}
            y={15 - height}
            width={3}
            height={height}
            rx={1}
          />
        ))}
      </svg>
    ),
    `performance-${level}`,
    { className: CSS.BM("performance-select-meter", level) },
  );

const LowMeter = createMeter("low");
const MediumMeter = createMeter("medium");
const HighMeter = createMeter("high");

const GLYPHS: Record<arc.task.Performance, Icon.ReactElement> = {
  auto: <Icon.Auto className={CSS.BE("performance-select", "auto")} />,
  low: <LowMeter />,
  medium: <MediumMeter />,
  high: <HighMeter />,
};

interface Level {
  key: arc.task.Performance;
  name: string;
  description: string;
}

const LEVELS: Level[] = [
  {
    key: "auto",
    name: "Auto",
    description: "Picks a strategy from the program's timers.",
  },
  {
    key: "low",
    name: "Low",
    description: "Least CPU. A fire can come a millisecond or more late.",
  },
  {
    key: "medium",
    name: "Medium",
    description: "Some CPU. Holds timing closely for most control loops.",
  },
  {
    key: "high",
    name: "High",
    description: "Up to a full CPU core. The most precise timing.",
  },
];

const ITEM_CLASS = CSS.BE("performance-select", "item");
const NAME_CLASS = CSS.BE("performance-select", "name");
const DESCRIPTION_CLASS = CSS.BE("performance-select", "description");

const ITEMS = LEVELS.map(({ key, name, description }) => (
  <Select.Item<arc.task.Performance>
    key={key}
    itemKey={key}
    className={ITEM_CLASS}
    y
    gap="tiny"
    align="start"
  >
    <Text.Text>
      {GLYPHS[key]}
      <span className={NAME_CLASS}>{name}</span>
    </Text.Text>
    <Text.Text className={DESCRIPTION_CLASS} color={9} level="small" wrap>
      {description}
    </Text.Text>
  </Select.Item>
));

const TRIGGER_PROPS = {
  className: CSS.BE("performance-select", "trigger"),
  collapsible: true,
};
const DIALOG_PROPS = { className: CSS.BE("performance-select", "dialog") };

export interface SelectPerformanceProps extends Omit<
  Select.SingleSimpleProps<arc.task.Performance>,
  | "children"
  | "resourceName"
  | "allowNone"
  | "variant"
  | "value"
  | "icon"
  | "triggerProps"
  | "dialogProps"
> {
  value: arc.task.Performance;
}

/**
 * Selects the performance level of an Arc task. The caller sets its width; when narrow
 * it shows only the glyph of the selected level.
 */
export const SelectPerformance = ({
  className,
  value,
  ...rest
}: SelectPerformanceProps): ReactElement => (
  <Select.Simple<arc.task.Performance>
    {...rest}
    value={value}
    className={CSS.cls(CSS.B("performance-select"), className)}
    allowNone={false}
    variant="floating"
    icon={GLYPHS[value]}
    triggerProps={TRIGGER_PROPS}
    dialogProps={DIALOG_PROPS}
    resourceName="performance"
  >
    {ITEMS}
  </Select.Simple>
);
