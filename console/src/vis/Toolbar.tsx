// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Status, Text } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

import { Toolbar } from "@/components";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Schematic } from "@/schematic";
import { Table } from "@/table";
import { createSelectorLayout } from "@/vis/Selector";
import { type LayoutType } from "@/vis/types";

interface ToolbarProps {
  layoutKey: string;
}

const TOOLBARS: Record<LayoutType, FC<ToolbarProps>> = {
  [LinePlot.LAYOUT_TYPE]: LinePlot.Toolbar,
  [Log.LAYOUT_TYPE]: Log.Toolbar,
  [Schematic.LAYOUT_TYPE]: Schematic.Toolbar,
  [Table.LAYOUT_TYPE]: Table.Toolbar,
};

const NoVis = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  const handleCreateNewVisualization = () => {
    placeLayout(createSelectorLayout());
  };
  return (
    <Align.Space justify="spaceBetween" style={{ height: "100%" }} empty>
      <Toolbar.Header>
        <Toolbar.Title icon={<Icon.Visualize />}>Visualization</Toolbar.Title>
      </Toolbar.Header>
      <Align.Center x size="small">
        <Status.Text level="p" variant="disabled" hideIcon>
          No visualization selected. Select a visualization or
        </Status.Text>
        <Text.Link level="p" onClick={handleCreateNewVisualization}>
          create a new one.
        </Text.Link>
      </Align.Center>
    </Align.Space>
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
  maxSize: 250,
  trigger: ["V"],
};
