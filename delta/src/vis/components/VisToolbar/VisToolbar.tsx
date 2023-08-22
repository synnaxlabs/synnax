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

import { ToolbarHeader } from "@/components";
import { NavDrawerItem, useLayoutPlacer, useSelectActiveMosaicLayout } from "@/layout";
import { LinePlotToolBar } from "@/line/controls/LinePlotToolbar";
import { PIDToolbar } from "@/pid/controls/PIDToolBar";
import { createVis } from "@/vis/core";

import { VisLayoutSelector } from "../VisLayoutSelector";

import { VisToolbarTitle } from "./VisToolbarTitle";

const SelectVis = ({ layoutKey }: { layoutKey?: string }): ReactElement => (
  <Align.Space justify="spaceBetween" style={{ height: "100%" }} empty>
    <ToolbarHeader>
      <VisToolbarTitle />
    </ToolbarHeader>
    <VisLayoutSelector layoutKey={layoutKey} />
  </Align.Space>
);

const NoVis = (): ReactElement => {
  const placer = useLayoutPlacer();
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
          onClick={() => placer(createVis({}))}
        >
          create a new one
        </Button.Button>
      </Align.Center>
    </Align.Space>
  );
};

const Content = (): ReactElement => {
  const layout = useSelectActiveMosaicLayout();
  switch (layout?.type) {
    case "pid":
      return <PIDToolbar layoutKey={layout?.key} />;
    case "line":
      return <LinePlotToolBar layoutKey={layout?.key} />;
    case "vis":
      return <SelectVis layoutKey={layout?.key} />;
    default:
      return <NoVis />;
  }
};

export const VisToolbar: NavDrawerItem = {
  key: "visualization",
  content: <Content />,
  tooltip: "Visualize",
  icon: <Icon.Visualize />,
  minSize: 125,
  maxSize: 250,
};
