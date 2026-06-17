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

import { EmptyAction, Toolbar } from "@/components";
import { Nav } from "@/nav";
import { Selector } from "@/selector";

import { useTabRenderer } from "./renderer";
import { useSelectFocused, useSelectSelected } from "./selectors";

const NoVis = (): ReactElement => {
  const handleCreateNewVisualization = () => {};
  const createComponentEnabled = Selector.useVisible();
  let message: string = "No visualization selected. Select a visualization";
  if (!createComponentEnabled) message += ".";
  else message += " or ";
  const action = createComponentEnabled ? "create a new one." : undefined;
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title icon={<Icon.Visualize />}>Visualization</Toolbar.Title>
      </Toolbar.Header>
      <EmptyAction
        x
        message={message}
        action={action}
        onClick={handleCreateNewVisualization}
      />
    </Toolbar.Content>
  );
};

const Content = (): ReactElement => {
  const Toolbar = useTabRenderer()?.Toolbar;
  if (Toolbar == null) return <NoVis />;
  return (
    <Errors.SuspenseBoundary>
      <Toolbar />
    </Errors.SuspenseBoundary>
  );
};

const Wrapper = () => {
  const panelKey = useSelectSelected();
  const tabKey = useSelectFocused();
  if (panelKey || tabKey == null) return <NoVis />;
  return (
    <Panel.KeyContext value={panelKey}>
      <Panel.TabKeyContext value={tabKey}>
        <Content />
      </Panel.TabKeyContext>
    </Panel.KeyContext>
  );
};

export const TOOLBAR: Nav.DrawerItem = {
  key: "visualization",
  content: <Wrapper />,
  tooltip: "Visualize",
  icon: <Icon.Visualize />,
  minSize: 160,
  maxSize: 300,
  trigger: ["V"],
};
