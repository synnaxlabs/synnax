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

const GLYPHS: Record<arc.task.Performance, Icon.ReactElement> = {
  auto: <Icon.Auto className={CSS.BE("performance-select", "auto")} />,
  low: <Icon.Meter.Low className={CSS.BM("performance-select-meter", "low")} />,
  medium: (
    <Icon.Meter.Medium className={CSS.BM("performance-select-meter", "medium")} />
  ),
  high: <Icon.Meter.High className={CSS.BM("performance-select-meter", "high")} />,
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
const LEVEL_CLASS = CSS.BE("performance-select", "level");
const NAME_CLASS = CSS.BE("performance-select", "name");
const DESCRIPTION_CLASS = CSS.BE("performance-select", "description");

// The trigger shows the selected level's glyph as its icon, so its label is the name.
const ITEMS = LEVELS.map(({ key, name, description }) => (
  <Select.Item<arc.task.Performance>
    key={key}
    itemKey={key}
    label={name}
    className={ITEM_CLASS}
    textValue={name}
  >
    <div className={LEVEL_CLASS}>
      {GLYPHS[key]}
      <span className={NAME_CLASS}>{name}</span>
      <Text.Text className={DESCRIPTION_CLASS} color={9} level="small" overflow="wrap">
        {description}
      </Text.Text>
    </div>
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
    searchHidden
  >
    {ITEMS}
  </Select.Simple>
);
