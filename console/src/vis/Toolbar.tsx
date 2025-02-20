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
import { type FC } from "react";

import { Toolbar as Core } from "@/components";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Schematic } from "@/schematic";
import { Table } from "@/table";
import { SELECTOR_LAYOUT } from "@/vis/Selector";
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

const NoVis = () => {
  const placeLayout = Layout.usePlacer();
  return (
    <Align.Space justify="spaceBetween" style={{ height: "100%" }} empty>
      <Core.Header>
        <Core.Title icon={<Icon.Visualize />}>Visualization</Core.Title>
      </Core.Header>
      <Align.Center direction="x" size="small">
        <Status.Text level="p" variant="disabled" hideIcon>
          No visualization selected. Select a visualization or
        </Status.Text>
        <Text.Link level="p" onClick={() => placeLayout(SELECTOR_LAYOUT)}>
          create a new one.
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};

const Content = () => {
  const layout = Layout.useSelectActiveMosaicLayout();
  if (layout == null) return <NoVis />;
  const Toolbar = TOOLBARS[layout.type as LayoutType];
  return Toolbar == null ? <NoVis /> : <Toolbar layoutKey={layout.key} />;
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "visualization",
  content: <Content />,
  tooltip: "Visualize",
  icon: <Icon.Visualize />,
  minSize: 125,
  maxSize: 250,
};
