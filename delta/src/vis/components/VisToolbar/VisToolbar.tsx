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
import { Button, Space, Status } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { VisLayoutSelector } from "../VisLayoutSelector";

import { VisToolbarTitle } from "./VisToolbarTitle";

import { ToolbarHeader } from "@/components";
import { NavDrawerItem, useLayoutPlacer, useSelectActiveMosaicLayout } from "@/layout";
import { LinePlotToolBar } from "@/line/controls/LinePlotToolbar";
import { PIDToolbar } from "@/pid/controls/PIDToolBar";
import { createVis } from "@/vis/core";

const SelectVis = ({ layoutKey }: { layoutKey?: string }): ReactElement => (
  <Space justify="spaceBetween" style={{ height: "100%" }} empty>
    <ToolbarHeader>
      <VisToolbarTitle />
    </ToolbarHeader>
    <VisLayoutSelector layoutKey={layoutKey} />;
  </Space>
);

const NoVis = (): ReactElement => {
  const placer = useLayoutPlacer();
  return (
    <Space justify="spaceBetween" style={{ height: "100%" }} empty>
      <ToolbarHeader>
        <VisToolbarTitle />
        <Space.Centered direction="x" size="small">
          <Status.Text level="p" variant="disabled" hideIcon>
            No visualization selected. Selecte a visualization or
          </Status.Text>
          <Button
            startIcon={<Icon.Add />}
            variant="outlined"
            onClick={() => placer(createVis({}))}
          >
            create a new one
          </Button>
        </Space.Centered>
      </ToolbarHeader>
    </Space>
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
  icon: <Icon.Visualize />,
  minSize: 125,
  maxSize: 250,
};
