// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs as Base } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { QUERY_ATTRIBUTE } from "@/components/tabs/sync";

// Astro's React SSR camelCases dashed slot names.
const slotName = (key: string): string =>
  key.replace(/[-_]([a-z])/g, (_, c: string) => c.toUpperCase());

export interface TabEntry {
  tabKey: string;
  name: string;
  icon?: ReactElement;
}

export interface TabsProps extends Record<string, ReactElement | any> {
  tabs: TabEntry[];
  /** Syncs the selected tab with every block on the page that shares the key. */
  queryParamKey?: string;
}

/**
 * Renders tabbed MDX slots on the server with the first tab selected. The tabs script
 * in `@/components/tabs/sync` switches them in the browser, so the component never
 * hydrates.
 */
export const Tabs = ({ tabs, queryParamKey, ...rest }: TabsProps): ReactElement => (
  <Base.Frame
    initialValue={tabs[0].tabKey}
    {...{ [QUERY_ATTRIBUTE]: queryParamKey ?? "" }}
  >
    <Base.Selector>
      {tabs.map(({ tabKey, name, icon }) => (
        <Base.Tab key={tabKey} itemKey={tabKey}>
          {icon}
          <Text.Text>{name}</Text.Text>
        </Base.Tab>
      ))}
    </Base.Selector>
    {tabs.map(({ tabKey }) => (
      <Base.Content key={tabKey} itemKey={tabKey} keepMounted>
        {rest[slotName(tabKey)]}
      </Base.Content>
    ))}
  </Base.Frame>
);
