// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Errors, Icon, Panel } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { useTab } from "@/platform/panel/tab";
import { Toolbar } from "@/platform/toolbar";
import { Session } from "@/session";

const EmptyContent = (): ReactElement => (
  <Toolbar.Content>
    <Toolbar.Header>
      <Toolbar.Title icon={<Icon.Visualize />}>Tab</Toolbar.Title>
    </Toolbar.Header>
    <Empty.Action x message="No tab selected." />
  </Toolbar.Content>
);

const Content = (): ReactElement => {
  const { Toolbar } = useTab();
  if (Toolbar == null) return <EmptyContent />;
  return (
    <Errors.SuspenseBoundary>
      <Toolbar />
    </Errors.SuspenseBoundary>
  );
};

const Wrapper = () => {
  const panelKey = Session.Panel.useSelectSelected();
  const tabKey = Session.Panel.useSelectFocusedTab(panelKey);
  if (panelKey == null || tabKey == null) return <EmptyContent />;
  return (
    <Panel.Scope.Provider value={panelKey}>
      <Panel.TabScope.Provider value={tabKey}>
        <Content />
      </Panel.TabScope.Provider>
    </Panel.Scope.Provider>
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "tab",
  content: <Wrapper />,
  tooltip: "Tab",
  icon: <Icon.Visualize />,
  sizeBounds: { lower: 160, upper: 300 },
  trigger: ["V"],
};
