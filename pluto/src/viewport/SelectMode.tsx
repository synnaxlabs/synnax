// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Mode, type UseTriggers } from "@/viewport/use";

export type FilteredMode = Exclude<Mode, "cancel">;

interface TooltipProps {
  mode: FilteredMode;
  triggers: Triggers.Trigger[];
}

export const TooltipText = ({ mode, triggers }: TooltipProps): ReactElement => (
  <Text.Text level="small">
    {caseconv.capitalize(mode)}
    <Triggers.Text trigger={triggers[0]} el="span" />
  </Text.Text>
);

export interface SelectModeProps
  extends Select.ButtonsProps<Mode>, Omit<Tooltip.ExtensionProps, "tooltip"> {
  triggers: UseTriggers;
}

export const SelectMode = ({
  triggers,
  value,
  onChange,
  tooltipLocation,
  hideTooltip,
  ...rest
}: SelectModeProps): ReactElement => {
  const commonProps: Partial<Select.ItemProps<Mode>> = {
    tooltipLocation,
    hideTooltip,
    size: "small",
  };
  return (
    <Select.Buttons variant="outlined" {...rest} value={value} onChange={onChange}>
      <Select.Item
        itemKey="zoom"
        tooltip={<TooltipText mode="zoom" triggers={triggers.modes.zoom} />}
        {...commonProps}
      >
        <Icon.Zoom />
      </Select.Item>
      <Select.Item
        itemKey="pan"
        tooltip={<TooltipText mode="pan" triggers={triggers.modes.pan} />}
        {...commonProps}
      >
        <Icon.Pan />
      </Select.Item>
      <Select.Item
        itemKey="select"
        tooltip={<TooltipText mode="select" triggers={triggers.modes.select} />}
        {...commonProps}
      >
        <Icon.Selection />
      </Select.Item>
    </Select.Buttons>
  );
};
