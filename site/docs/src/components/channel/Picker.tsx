// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs as BaseTabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { Tabs } from "@/components/tabs";

export interface Option {
  type: string;
  name: string;
  /** The rendered SVG of the icon of the channel type. */
  icon?: string;
  /** The rendered HTML of the channel type. */
  pane: string;
}

export interface Group {
  key: string;
  name: string;
  /** The rendered SVG of the icon of the group. */
  icon?: string;
  options: Option[];
}

const RenderedIcon = ({ svg }: { svg?: string }): ReactElement | undefined =>
  svg == null ? undefined : (
    <span className="channel-types__icon" dangerouslySetInnerHTML={{ __html: svg }} />
  );

export interface OptionsProps {
  options: Option[];
}

/** Renders a wrapping set of chips for channel types, and the selected type. */
export const Options = ({ options }: OptionsProps): ReactElement => (
  <BaseTabs.Frame
    className="channel-types__options"
    initialValue={options[0].type}
    {...{ [Tabs.QUERY_ATTRIBUTE]: "" }}
  >
    <BaseTabs.Selector variant="chip" size="small" wrap>
      {options.map(({ type, name, icon }) => (
        <BaseTabs.Tab key={type} itemKey={type}>
          <RenderedIcon svg={icon} />
          <Text.Text>{name}</Text.Text>
        </BaseTabs.Tab>
      ))}
    </BaseTabs.Selector>
    {options.map(({ type, pane }) => (
      <BaseTabs.Content key={type} itemKey={type} keepMounted>
        <div dangerouslySetInnerHTML={{ __html: pane }} />
      </BaseTabs.Content>
    ))}
  </BaseTabs.Frame>
);

export interface PickerProps {
  groups: Group[];
}

/** Renders a tab set of groups, and the {@link Options} of each group, in one pane. */
export const Picker = ({ groups }: PickerProps): ReactElement => (
  <Tabs.Tabs
    tabs={groups.map(({ key, name, icon }) => ({
      tabKey: key,
      name,
      icon: <RenderedIcon svg={icon} />,
    }))}
    {...Object.fromEntries(
      groups.map(({ key, options }) => [
        Tabs.slotName(key),
        <Options key={key} options={options} />,
      ]),
    )}
  />
);
