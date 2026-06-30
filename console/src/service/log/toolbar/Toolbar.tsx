// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/log/toolbar/Toolbar.css";

import { log } from "@synnaxlabs/client";
import { Flex, Icon, Log, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Base } from "@/component";
import { CSS } from "@/component/css";
import { Export } from "@/export";
import { useExport } from "@/service/log/imex/export";
import { Channels } from "@/service/log/toolbar/Channels";
import { Properties } from "@/service/log/toolbar/Properties";
import { Session } from "@/session";

const TABS: Tabs.Tab[] = [
  { tabKey: "channels", name: "Channels" },
  { tabKey: "properties", name: "Properties" },
];

const Internal = (): ReactElement => {
  const dispatch = useDispatch();
  const selected = Session.Log.useSelectSelectedToolbarTab();
  const name = Log.useSelectName();
  const key = Log.useKey();
  const handleTabSelect = useCallback(
    (tab: string) =>
      dispatch(
        Session.Log.setSelectedToolbarTab({
          key,
          tab: tab as Session.Log.ToolbarTab,
        }),
      ),
    [dispatch, key],
  );
  const handleExport = useExport();

  const content = useCallback(({ tabKey }: Tabs.Tab) => {
    switch (tabKey) {
      case "properties":
        return <Properties />;
      default:
        return <Channels />;
    }
  }, []);

  const tabsValue = useMemo(
    () => ({ tabs: TABS, selected, content, onSelect: handleTabSelect }),
    [selected, content, handleTabSelect],
  );

  return (
    <Base.Content className={CSS.B("log-toolbar")}>
      <Tabs.Provider value={tabsValue}>
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

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => (
  <Log.Suspended logKey={layoutKey}>
    <Internal />
  </Log.Suspended>
);
