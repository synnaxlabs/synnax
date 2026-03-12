// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Flex, Icon, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo, useState } from "react";

import { Cluster } from "@/cluster";
import { Toolbar as Base } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { useExport } from "@/log/export";
import { useSelectOptional } from "@/log/selectors";
import { Channels } from "@/log/toolbar/Channels";
import { Text } from "@/log/toolbar/Text";

export interface ToolbarProps {
  layoutKey: string;
}

const TABS: Tabs.Tab[] = [
  { tabKey: "channels", name: "Channels" },
  { tabKey: "text", name: "Text" },
];

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const state = useSelectOptional(layoutKey);
  const [activeTab, setActiveTab] = useState("channels");
  const handleExport = useExport();

  const content = useCallback(
    ({ tabKey }: Tabs.Tab) => {
      if (tabKey === "text") return <Text layoutKey={layoutKey} />;
      return <Channels layoutKey={layoutKey} />;
    },
    [layoutKey],
  );

  const tabsValue = useMemo(
    () => ({ tabs: TABS, selected: activeTab, content, onSelect: setActiveTab }),
    [activeTab, content],
  );

  if (state == null) return null;
  return (
    <Base.Content>
      <Tabs.Provider value={tabsValue}>
        <Base.Header>
          <Base.Title icon={<Icon.Log />}>{name}</Base.Title>
          <Flex.Box x align="center" empty>
            <Flex.Box x empty style={{ width: 66 }}>
              <Export.ToolbarButton onExport={() => handleExport(state.key)} />
              <Cluster.CopyLinkToolbarButton
                name={name}
                ontologyID={log.ontologyID(state.key)}
              />
            </Flex.Box>
            <Tabs.Selector style={{ borderBottom: "none" }} />
          </Flex.Box>
        </Base.Header>
        <Tabs.Content />
      </Tabs.Provider>
    </Base.Content>
  );
};
