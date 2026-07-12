// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs as Base } from "@synnaxlabs/pluto/tabs";
import { type ReactElement, useEffect, useState } from "react";

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

  const handleSelect = (tabKey: string) => {
    if (queryParamKey == null) return setSelected(tabKey);
    const url = new URL(window.location.href);
    url.searchParams.set(queryParamKey, tabKey);
    window.history.pushState({}, "", url.toString());
    window.dispatchEvent(new CustomEvent("urlchange"));
    setSelected(tabKey);
  };

  const getSelected = (): string => {
    if (queryParamKey == null) return selected;
    const url = new URL(window.location.href);
    return url.searchParams.get(queryParamKey) ?? selected;
  };

  useEffect(() => {
    const updateFromURL = () => {
      if (queryParamKey == null) return;
      const url = new URL(window.location.href);
      setSelected(url.searchParams.get(queryParamKey) ?? tabs[0].tabKey);
    };
    handleSelect(getSelected());
    window.addEventListener("popstate", updateFromURL);
    window.addEventListener("urlchange", updateFromURL);
    return () => {
      window.removeEventListener("popstate", updateFromURL);
      window.removeEventListener("urlchange", updateFromURL);
    };
  }, [queryParamKey]);

  return (
    <Base.Frame value={getSelected()} onChange={handleSelect}>
      <Base.Selector>
        {tabs.map(({ tabKey, name, icon }) => (
          <Base.Tab key={tabKey} itemKey={tabKey}>
            {icon ?? rest[`${tabKey}-icon`]}
            {name}
          </Base.Tab>
        ))}
      </Base.Selector>
      {tabs.map(({ tabKey }) => (
        <Base.Content key={tabKey} itemKey={tabKey}>
          {rest[tabKey]}
        </Base.Content>
      ))}
    </Base.Frame>
  );
};
