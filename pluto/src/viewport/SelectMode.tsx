// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { type Icon as PIcon } from "@/icon";
import { Select } from "@/select";
import { Text } from "@/text";
import { Triggers } from "@/triggers";
import { type Trigger } from "@/triggers/triggers";
import { type Mode, MODES, type UseTriggers } from "@/viewport/use";

interface Entry {
  key: Mode;
  icon: PIcon.Element;
  tooltip: ReactNode;
}

interface TooltipProps {
  mode: Mode;
  triggers: Trigger[];
}

const Tooltip = ({ mode, triggers }: TooltipProps): ReactElement => (
  <Align.Space direction="x" align="center">
    <Text.Text level="small">{caseconv.capitalize(mode)}</Text.Text>
    <Align.Space empty direction="x" align="center">
      <Triggers.Text trigger={triggers[0]} level="small" />
    </Align.Space>
  </Align.Space>
);

const MODE_ICONS: Record<Mode, ReactElement> = {
  zoom: <Icon.Zoom />,
  pan: <Icon.Pan />,
  select: <Icon.Selection />,
  zoomReset: <Icon.Expand />,
  click: <Icon.Bolt />,
};

export interface SelectModeProps extends Omit<Select.ButtonProps<Mode>, "data"> {
  triggers: UseTriggers;
  disable?: Mode[];
}

export const SelectMode = ({
  triggers,
  disable = ["zoomReset", "click"],
  ...rest
}: SelectModeProps): ReactElement => {
  const data = Object.entries(triggers)
    .filter(([key]) => !disable.includes(key as Mode) && MODES.includes(key as Mode))
    .map(([key, value]) => ({
      key: key as Mode,
      icon: MODE_ICONS[key as Mode],
      tooltip: <Tooltip mode={key as Mode} triggers={value as Trigger[]} />,
    }))
    .sort((a, b) => MODES.indexOf(a.key) - MODES.indexOf(b.key)) as Entry[];

  return (
    <Select.Button<Mode, Entry> {...rest} data={data} entryRenderKey="icon">
      {({ title: _, entry, ...rest }) => (
        <Button.Icon
          {...rest}
          key={entry.key}
          variant={rest.selected ? "filled" : "text"}
          size="medium"
          tooltip={entry.tooltip}
          tooltipLocation={{ x: "right", y: "top" }}
        >
          {entry.icon}
        </Button.Icon>
      )}
    </Select.Button>
  );
};
