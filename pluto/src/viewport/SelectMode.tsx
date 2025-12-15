// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { Icon } from "@/icon";
import { Select } from "@/select";
import { Text } from "@/text";
import { type Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";
import { type Trigger } from "@/triggers/triggers";
import { type Mode, MODES, type UseTriggers } from "@/viewport/use";

export type FilteredMode = Exclude<Mode, "cancel">;

interface TooltipProps {
  mode: FilteredMode;
  triggers: Trigger[];
}

export const TooltipText = ({ mode, triggers }: TooltipProps): ReactElement => (
  <Text.Text level="small">
    {caseconv.capitalize(mode)}
    <Triggers.Text trigger={triggers[0]} el="span" />
  </Text.Text>
);

export interface SelectModeProps
  extends Omit<Select.ButtonsProps<Mode>, "keys">, Omit<Tooltip.WrapProps, "tooltip"> {
  triggers: UseTriggers;
  disable?: Mode[];
}

export const SelectMode = ({
  triggers,
  value,
  onChange,
  disable = ["zoomReset", "click", "cancel"],
  tooltipDelay,
  tooltipLocation,
  hideTooltip,
  ...rest
}: SelectModeProps): ReactElement => {
  const data = useMemo(() => MODES.filter((m) => !disable.includes(m)), [disable]);
  const commonProps: Partial<Select.ButtonProps<Mode>> = {
    tooltipDelay,
    tooltipLocation,
    hideTooltip,
    size: "small",
  };
  return (
    <Select.Buttons {...rest} keys={data} value={value} onChange={onChange}>
      <Select.Button
        itemKey="zoom"
        tooltip={<TooltipText mode="zoom" triggers={triggers.zoom} />}
        {...commonProps}
      >
        <Icon.Zoom />
      </Select.Button>
      <Select.Button
        itemKey="pan"
        tooltip={<TooltipText mode="pan" triggers={triggers.pan} />}
        {...commonProps}
      >
        <Icon.Pan />
      </Select.Button>
      <Select.Button
        itemKey="select"
        tooltip={<TooltipText mode="select" triggers={triggers.select} />}
        {...commonProps}
      >
        <Icon.Selection />
      </Select.Button>
    </Select.Buttons>
  );
};
