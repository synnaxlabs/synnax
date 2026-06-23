// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layered/service/log/toolbar/Toolbar.css";

import { log } from "@synnaxlabs/client";
import { Flex, Icon, Log, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Base } from "@/components";
import { CSS } from "@/css";
import { Export } from "@/export";
import { useExport } from "@/layered/service/log/imex/export";
import { Channels } from "@/layered/service/log/toolbar/Channels";
import { Properties } from "@/layered/service/log/toolbar/Properties";
import { Session } from "@/layered/session";
import { Layout } from "@/layout";

export interface ToolbarProps {
  layoutKey: string;
}

const TABS: Tabs.Tab[] = [
  { tabKey: "channels", name: "Channels" },
  { tabKey: "properties", name: "Properties" },
];

const Internal = ({ layoutKey }: ToolbarProps): ReactElement => {
  Log.useEnsureRetrieved({ key: layoutKey });
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const activeTab = Session.Log.useSelectActiveToolbarTab({ key: layoutKey });
  const handleTabSelect = useCallback(
    (tab: string) =>
      dispatch(
        Session.Log.setActiveToolbarTab({
          key: layoutKey,
          tab: tab as Session.Log.ToolbarTab,
        }),
      ),
    [dispatch, layoutKey],
  );
  const handleExport = useExport();

  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      switch (tabKey) {
        case "properties":
          return <Properties layoutKey={layoutKey} />;
        default:
          return <Channels layoutKey={layoutKey} />;
      }
    },
    [layoutKey],
  );

  const tabsValue = useMemo(
    () => ({ tabs: TABS, selected: activeTab, content, onSelect: handleTabSelect }),
    [activeTab, content, handleTabSelect],
  );

  return (
    <Base.Content className={CSS.B("log-toolbar")}>
      <Tabs.Provider value={tabsValue}>
        <Base.Header>
          <Base.Title icon={<Icon.Log />}>{name}</Base.Title>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty className={CSS.BE("log-toolbar", "actions")}>
              <Export.ToolbarButton onExport={() => handleExport(layoutKey)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={log.ontologyID(layoutKey)}
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

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const exists = Session.Log.useSelectExists({ key: layoutKey });
  if (!exists) return null;
  return <Internal layoutKey={layoutKey} />;
};
