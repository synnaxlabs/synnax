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
  const compensationFrame = useRef(0);

  // Synced blocks above this one resize on select, each in its own React root on
  // its own schedule. Scroll away the drift over two frames to keep this block put.
  const compensateScroll = () => {
    const el = frameRef.current;
    if (el == null) return;
    const top = el.getBoundingClientRect().top;
    const compensate = () => {
      const delta = el.getBoundingClientRect().top - top;
      if (delta !== 0) window.scrollBy(0, delta);
    };
    cancelAnimationFrame(compensationFrame.current);
    compensationFrame.current = requestAnimationFrame(() => {
      compensate();
      compensationFrame.current = requestAnimationFrame(compensate);
    });
  };

  useEffect(() => () => cancelAnimationFrame(compensationFrame.current), []);

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
    const updateFromURL = () => {
      const url = new URL(window.location.href);
      setSelected(url.searchParams.get(queryParamKey) ?? tabs[0].tabKey);
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
        <Base.Content key={tabKey} itemKey={tabKey}>
          {rest[tabKey] ?? rest[slotName(tabKey)]}
        </Base.Content>
      ))}
    </Base.Frame>
  );
};
