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

interface MeterProps {
  level: ManualPerformance;
}

/** Shows a level as three rising bars, filled up to the level. */
const Meter = ({ level }: MeterProps): ReactElement => (
  <svg
    className={CSS.cls(
      CSS.BE("performance-select", "meter"),
      CSS.BM("performance-select-meter", level),
    )}
    viewBox="0 0 16 16"
    aria-hidden
  >
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
);

const AUTO_CLASS = CSS.BE("performance-select", "auto");

interface Level {
  key: arc.task.Performance;
  glyph: ReactElement;
  name: string;
  description: string;
}

const LEVELS: Level[] = [
  {
    key: "auto",
    glyph: <Icon.Auto className={AUTO_CLASS} />,
    name: "Auto",
    description: "Picks a strategy from the program's timers.",
  },
  {
    key: "low",
    glyph: <Meter level="low" />,
    name: "Low",
    description: "Least CPU. Timing can drift by a millisecond or more.",
  },
  {
    key: "medium",
    glyph: <Meter level="medium" />,
    name: "Medium",
    description: "Some CPU. Holds timing closely for most control loops.",
  },
  {
    key: "high",
    glyph: <Meter level="high" />,
    name: "High",
    description: "Up to a full CPU core. The most precise timing.",
  },
];

const ITEM_CLASS = CSS.BE("performance-select", "item");
const NAME_CLASS = CSS.BE("performance-select", "name");
const DESCRIPTION_CLASS = CSS.BE("performance-select", "description");

const ITEMS = LEVELS.map(({ key, glyph, name, description }) => (
  <Select.Item<arc.task.Performance>
    key={key}
    itemKey={key}
    className={ITEM_CLASS}
    y
    gap="tiny"
    align="start"
  >
    <Text.Text>
      {glyph}
      <span className={NAME_CLASS}>{name}</span>
    </Text.Text>
    <Text.Text className={DESCRIPTION_CLASS} color={9} level="small" wrap>
      {description}
    </Text.Text>
  </Select.Item>
));

const TRIGGER_PROPS = { className: CSS.BE("performance-select", "trigger") };
const DIALOG_PROPS = { className: CSS.BE("performance-select", "dialog") };

export interface SelectPerformanceProps extends Omit<
  Select.SingleSimpleProps<arc.task.Performance>,
  "children" | "resourceName" | "allowNone" | "variant" | "triggerProps" | "dialogProps"
> {}

/** Selects the performance level of an Arc task. */
export const SelectPerformance = ({
  className,
  ...rest
}: SelectPerformanceProps): ReactElement => (
  <Select.Simple<arc.task.Performance>
    {...rest}
    className={CSS.cls(CSS.B("performance-select"), className)}
    allowNone={false}
    variant="floating"
    triggerProps={TRIGGER_PROPS}
    dialogProps={DIALOG_PROPS}
    resourceName="performance"
  >
    {ITEMS}
  </Select.Simple>
);
