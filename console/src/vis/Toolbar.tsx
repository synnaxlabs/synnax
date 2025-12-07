// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

import { Arc } from "@/arc";
import { EmptyAction, Toolbar } from "@/components";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Schematic } from "@/schematic";
import { Table } from "@/table";
import { createSelectorLayout, useSelectorVisible } from "@/vis/Selector";
import { type LayoutType } from "@/vis/types";

interface ToolbarProps {
  layoutKey: string;
}

const TOOLBARS: Record<LayoutType, FC<ToolbarProps>> = {
  [LinePlot.LAYOUT_TYPE]: LinePlot.Toolbar,
  [Log.LAYOUT_TYPE]: Log.Toolbar,
  [Schematic.LAYOUT_TYPE]: Schematic.Toolbar,
  [Table.LAYOUT_TYPE]: Table.Toolbar,
  [Arc.Editor.LAYOUT_TYPE]: Arc.Editor.Toolbar,
};

const NoVis = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  const handleCreateNewVisualization = () => {
    placeLayout(createSelectorLayout());
  };
  const createComponentEnabled = useSelectorVisible();
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
  const layout = Layout.useSelectActiveMosaicLayout();
  if (layout == null) return <NoVis />;
  const Toolbar = TOOLBARS[layout.type as LayoutType];
  return Toolbar == null ? <NoVis /> : <Toolbar layoutKey={layout.key} />;
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "visualization",
  content: <Content />,
  tooltip: "Visualize",
  icon: <Icon.Visualize />,
  minSize: 160,
  maxSize: 300,
  trigger: ["V"],
};
