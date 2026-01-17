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

export interface TabsProps extends Record<string, ReactElement | any> {
  tabs: Base.Tab[];
  queryParamKey?: string;
}

export const Tabs = ({ tabs, queryParamKey, ...rest }: TabsProps): ReactElement => {
  tabs = tabs.map((tab) => ({
    ...tab,
    icon: tab.icon ?? rest[`${tab.tabKey}-icon`],
  }));
  const [selected, setSelected] = useState<string>(tabs[0].tabKey);

  const handleSelect = (tabKey: string) => {
    if (queryParamKey == null) return setSelected(tabKey);
    const url = new URL(window.location.href);
    url.searchParams.set(queryParamKey, tabKey);
    window.history.pushState({}, "", url.toString());
    setSelected(tabKey);
  };

  const getSelected = (): string => {
    if (queryParamKey == null) return selected;
    const url = new URL(window.location.href);
    return url.searchParams.get(queryParamKey) ?? selected;
  };

  useEffect(() => {
    handleSelect(getSelected());
    const i = setInterval(() => {
      if (queryParamKey == null) return;
      const url = new URL(window.location.href);
      setSelected(url.searchParams.get(queryParamKey) ?? tabs[0].tabKey);
    }, 200);
    return () => clearInterval(i);
  }, [queryParamKey]);

  const staticProps = Base.useStatic({
    selected,
    onSelect: handleSelect,
    tabs,
  });

  return (
    <Base.Tabs {...staticProps}>{(tab) => <div>{rest[tab.tabKey]}</div>}</Base.Tabs>
  );
};
