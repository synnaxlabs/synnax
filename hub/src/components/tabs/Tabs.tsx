// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs as Base } from "@synnaxlabs/pluto/tabs";
import { Text } from "@synnaxlabs/pluto/text";
import { type ReactElement, useEffect, useRef, useState } from "react";

// Astro's React SSR camelCases dashed slot names; hydration passes them raw.
const slotName = (key: string): string =>
  key.replace(/[-_]([a-z])/g, (_, c: string) => c.toUpperCase());

// Islands and media in the panels above keep landing for about this long.
const SETTLE_MS = 1000;
const READER_SCROLL_EVENTS = ["wheel", "touchstart", "keydown"];

export interface TabEntry {
  tabKey: string;
  name: string;
  icon?: ReactElement;
}

export interface TabsProps extends Record<string, ReactElement | any> {
  tabs: TabEntry[];
  queryParamKey?: string;
}

export const Tabs = ({ tabs, queryParamKey, ...rest }: TabsProps): ReactElement => {
  const [selected, setSelected] = useState<string>(tabs[0].tabKey);
  const frameRef = useRef<HTMLDivElement>(null);
  const settling = useRef<AbortController>(null);

  // Synced blocks above this one resize after a select, so scroll their drift away.
  const compensateScroll = () => {
    const el = frameRef.current;
    if (el == null) return;
    settling.current?.abort();
    const controller = new AbortController();
    settling.current = controller;
    const { signal } = controller;
    const top = el.getBoundingClientRect().top;
    const observer = new ResizeObserver(() => {
      const delta = el.getBoundingClientRect().top - top;
      if (delta !== 0) window.scrollBy(0, delta);
    });
    observer.observe(document.body);
    const stop = () => controller.abort();
    for (const event of READER_SCROLL_EVENTS)
      window.addEventListener(event, stop, { signal });
    const timer = setTimeout(stop, SETTLE_MS);
    signal.addEventListener("abort", () => {
      observer.disconnect();
      clearTimeout(timer);
    });
  };

  useEffect(() => () => settling.current?.abort(), []);

  const handleSelect = (tabKey: string) => {
    compensateScroll();
    setSelected(tabKey);
    if (queryParamKey == null) return;
    const url = new URL(window.location.href);
    url.searchParams.set(queryParamKey, tabKey);
    window.history.pushState({}, "", url.toString());
    window.dispatchEvent(new CustomEvent("urlchange"));
  };

  useEffect(() => {
    if (queryParamKey == null) return;
    // A block without a tab for the key keeps the tab it already shows.
    const updateFromURL = () => {
      const url = new URL(window.location.href);
      const key = url.searchParams.get(queryParamKey) ?? tabs[0].tabKey;
      if (tabs.some((tab) => tab.tabKey === key)) setSelected(key);
    };
    updateFromURL();
    window.addEventListener("popstate", updateFromURL);
    window.addEventListener("urlchange", updateFromURL);
    return () => {
      window.removeEventListener("popstate", updateFromURL);
      window.removeEventListener("urlchange", updateFromURL);
    };
  }, [queryParamKey]);

  return (
    <Base.Frame ref={frameRef} value={selected} onChange={handleSelect}>
      <Base.Selector>
        {tabs.map(({ tabKey, name, icon }) => (
          <Base.Tab key={tabKey} itemKey={tabKey}>
            {icon ?? rest[`${tabKey}-icon`]}
            <Text.Text>{name}</Text.Text>
          </Base.Tab>
        ))}
      </Base.Selector>
      {tabs.map(({ tabKey }) => (
        <Base.Content key={tabKey} itemKey={tabKey} keepMounted>
          {rest[tabKey] ?? rest[slotName(tabKey)]}
        </Base.Content>
      ))}
    </Base.Frame>
  );
};
