// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Button, Align, Status } from "@synnaxlabs/pluto";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { PID } from "@/pid";
import { create } from "@/vis/create";

import { LayoutSelector } from "./LayoutSelector";

export const VisToolbarTitle = (): ReactElement => (
  <ToolbarTitle icon={<Icon.Visualize />}>Visualization</ToolbarTitle>
);

const SelectVis = ({ layoutKey }: { layoutKey?: string }): ReactElement => (
  <Align.Space justify="spaceBetween" style={{ height: "100%" }} empty>
    <ToolbarHeader>
      <VisToolbarTitle />
    </ToolbarHeader>
    <LayoutSelector layoutKey={layoutKey} />
  </Align.Space>
);

const NoVis = (): ReactElement => {
  const placer = Layout.usePlacer();
  return (
    <Align.Space justify="spaceBetween" style={{ height: "100%" }} empty>
      <ToolbarHeader>
        <VisToolbarTitle />
      </ToolbarHeader>
      <Align.Center direction="x" size="small">
        <Status.Text level="p" variant="disabled" hideIcon>
          No visualization selected. Selecte a visualization or
        </Status.Text>
        <Button.Button
          startIcon={<Icon.Add />}
          variant="outlined"
          onClick={() => placer(create({}))}
        >
          create a new one
        </Button.Button>
      </Align.Center>
    </Align.Space>
  );
};

const Content = (): ReactElement => {
  const layout = Layout.useSelectActiveMosaicLayout();
  switch (layout?.type) {
    case "pid":
      return <PID.Toolbar layoutKey={layout?.key} />;
    case "line":
      return <LinePlot.Toolbar layoutKey={layout?.key} />;
    case "vis":
      return <SelectVis layoutKey={layout?.key} />;
    default:
      return <NoVis />;
  }
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "visualization",
  content: <Content />,
  tooltip: "Visualize",
  icon: <Icon.Visualize />,
  minSize: 125,
  maxSize: 250,
};
