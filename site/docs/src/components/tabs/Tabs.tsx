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

export interface TabEntry<K extends string = string> {
  tabKey: K;
  name: string;
  icon?: ReactElement;
}

/** The props an MDX page passes: one slot per tab, and the tab filters. */
export interface FilterProps<K extends string = string> extends Record<
  string,
  ReactElement | any
> {
  exclude?: K[];
  /** Tabs to show first, in this order. The other tabs keep their order. */
  priority?: K[];
}

export interface TabsProps<K extends string = string> extends FilterProps<K> {
  tabs: TabEntry<K>[];
  /** Syncs the selected tab with every block on the page that shares the key. */
  queryParamKey?: string;
}

/**
 * Renders tabbed MDX slots on the server with the first tab selected. The tabs script
 * in `@/components/tabs/sync` switches them in the browser, so the component never
 * hydrates.
 */
export const Tabs = <K extends string>({
  tabs,
  queryParamKey,
  exclude = [],
  priority = [],
  ...rest
}: TabsProps<K>): ReactElement => {
  const rank = (key: K): number => {
    const index = priority.indexOf(key);
    return index === -1 ? priority.length : index;
  };
  const shown = tabs
    .filter(({ tabKey }) => !exclude.includes(tabKey))
    .sort((a, b) => rank(a.tabKey) - rank(b.tabKey));
  return (
    <Base.Frame
      initialValue={shown[0].tabKey}
      {...{ [QUERY_ATTRIBUTE]: queryParamKey ?? "" }}
    >
      <Base.Selector>
        {shown.map(({ tabKey, name, icon }) => (
          <Base.Tab key={tabKey} itemKey={tabKey}>
            {icon}
            <Text.Text>{name}</Text.Text>
          </Base.Tab>
        ))}
      </Base.Selector>
      {shown.map(({ tabKey }) => (
        <Base.Content key={tabKey} itemKey={tabKey} keepMounted>
          {rest[slotName(tabKey)]}
        </Base.Content>
      ))}
    </Base.Frame>
  );
};
