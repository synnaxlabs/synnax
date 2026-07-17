// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/log/toolbar/Toolbar.css";

import { log } from "@synnaxlabs/client";
import { Flex, Icon, Log, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useExport } from "@/feature/log/export";
import { Channels } from "@/feature/log/toolbar/Channels";
import { Properties } from "@/feature/log/toolbar/Properties";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Export } from "@/platform/export";
import { Toolbar as Base } from "@/platform/toolbar";
import { Session } from "@/session";

const Internal = (): ReactElement => {
  const dispatch = Session.useDispatch();
  const selected = Session.Log.useSelectSelectedToolbarTab();
  const name = Log.useSelectName();
  const key = Log.useKey();
  const handleExport = useExport();
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

  return (
    <Base.Content className={CSS.B("log-toolbar")}>
      <Tabs.Frame value={selected} onChange={handleTabSelect} grow>
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
            <Tabs.Selector className={CSS.BE("log-toolbar", "tabs")}>
              <Tabs.Tab itemKey="channels">Channels</Tabs.Tab>
              <Tabs.Tab itemKey="properties">Properties</Tabs.Tab>
            </Tabs.Selector>
          </Flex.Box>
        </Base.Header>
        <Tabs.Content itemKey="channels">
          <Channels />
        </Tabs.Content>
        <Tabs.Content itemKey="properties">
          <Properties />
        </Tabs.Content>
      </Tabs.Frame>
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
