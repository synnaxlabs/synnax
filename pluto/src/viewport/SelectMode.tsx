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

import { Align } from "@/align";
import { Icon } from "@/icon";
import { Select } from "@/select";
import { Text } from "@/text";
import { Triggers } from "@/triggers";
import { type Trigger } from "@/triggers/triggers";
import { type Mode, MODES, type UseTriggers } from "@/viewport/use";

export type FilteredMode = Exclude<Mode, "cancel">;

interface TooltipProps {
  mode: FilteredMode;
  triggers: Trigger[];
}

const Tooltip = ({ mode, triggers }: TooltipProps): ReactElement => (
  <Align.Space x align="center">
    <Text.Text level="small">{caseconv.capitalize(mode)}</Text.Text>
    <Align.Space empty x align="center">
      <Triggers.Text trigger={triggers[0]} level="small" />
    </Align.Space>
  </Align.Space>
);

export interface SelectModeProps extends Select.ButtonsProps<Mode> {
  triggers: UseTriggers;
  disable?: Mode[];
}

export const SelectMode = ({
  triggers,
  value,
  onChange,
  disable = ["zoomReset", "click", "cancel"],
  ...rest
}: SelectModeProps): ReactElement => {
  const data = useMemo(() => MODES.filter((m) => !disable.includes(m)), [disable]);
  return (
    <Select.Buttons {...rest} keys={data} value={value} onChange={onChange}>
      <Select.ButtonIcon
        itemKey="zoom"
        tooltip={<Tooltip mode="zoom" triggers={triggers.zoom} />}
      >
        <Icon.Zoom />
      </Select.ButtonIcon>
      <Select.ButtonIcon
        itemKey="pan"
        tooltip={<Tooltip mode="pan" triggers={triggers.pan} />}
      >
        <Icon.Pan />
      </Select.ButtonIcon>
      <Select.ButtonIcon
        itemKey="select"
        tooltip={<Tooltip mode="select" triggers={triggers.select} />}
      >
        <Icon.Selection />
      </Select.ButtonIcon>
      <Select.ButtonIcon
        itemKey="zoomReset"
        tooltip={<Tooltip mode="zoomReset" triggers={triggers.zoomReset} />}
      >
        <Icon.Expand />
      </Select.ButtonIcon>
      <Select.ButtonIcon itemKey="click">
        <Icon.Bolt />
      </Select.ButtonIcon>
    </Select.Buttons>
  );
};
