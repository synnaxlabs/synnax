// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/log/toolbar/Toolbar.css";

import { log } from "@synnaxlabs/client";
import { Flex, Icon, Log, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { Cluster } from "@/cluster";
import { Toolbar as Base } from "@/components";
import { CSS } from "@/css";
import { Export } from "@/export";
import { useExport } from "@/log/export";
import { Session } from "@/log/session";
import { Tab } from "@/log/tab";
import { Channels } from "@/log/toolbar/Channels";
import { Properties } from "@/log/toolbar/Properties";

const TABS: Tabs.Tab[] = [
  { tabKey: "channels", name: "Channels" },
  { tabKey: "properties", name: "Properties" },
];

const content = ({ tabKey }: Tabs.Tab): ReactElement => {
  if (tabKey === "properties") return <Properties />;
  return <Channels />;
};

const Internal = (): ReactElement => {
  const key = Log.useKey();
  const name = Log.useSelectName({ key });
  const activeTab = Session.useSelectActiveToolbarTab();
  const handleSelectTab = Session.useSetActiveToolbarTab();
  const handleExport = useExport();

  const value = useMemo<Tabs.ContextValue>(
    () => ({
      tabs: TABS,
      selected: activeTab,
      onSelect: (tab) => handleSelectTab(tab as Session.ToolbarTab),
      content,
    }),
    [activeTab, handleSelectTab],
  );

  return (
    <Base.Content className={CSS.B("log-toolbar")}>
      <Tabs.Provider value={value}>
        <Base.Header>
          <Base.Title icon={<Icon.Log />}>{name}</Base.Title>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty className={CSS.BE("log-toolbar", "actions")}>
              <Export.ToolbarButton onExport={() => handleExport(key)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={log.ontologyID(key)}
              />
            </Flex.Box>
            <Tabs.Selector className={CSS.BE("log-toolbar", "tabs")} />
          </Flex.Box>
        </Base.Header>
        <Tabs.Content />
      </Tabs.Provider>
    </Base.Content>
  );
};

export const Toolbar = (): ReactElement => {
  const { key } = Tab.useArgs();
  return (
    <Log.Suspended logKey={key}>
      <Internal />
    </Log.Suspended>
  );
};
