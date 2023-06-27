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
import { Space, Status } from "@synnaxlabs/pluto";

import { VisToolbarTitle } from "./VisToolbarTitle";

import { ToolbarHeader } from "@/components";
import { NavDrawerItem } from "@/layout";
import { LinePlotToolBar } from "@/vis/line/controls/LinePlotToolbar";
import { PIDToolbar } from "@/vis/pid/controls/PIDToolBar";
import { useSelectVisMeta } from "@/vis/store";

const NoVisContent = (): ReactElement => (
  <Space justify="spaceBetween" style={{ height: "100%" }} empty>
    <ToolbarHeader>
      <VisToolbarTitle />
    </ToolbarHeader>
    <Status.Text.Centered level="h4" variant="disabled" hideIcon>
      No active visualization. Select a tab or create a new one.
    </Status.Text.Centered>
  </Space>
);

const Content = (): ReactElement => {
  const vis = useSelectVisMeta();
  switch (vis?.variant) {
    default:
      return <PIDToolbar layoutKey={vis?.key} />;
    // case "line":
    //   return <LinePlotToolBar layoutKey={vis?.key} />;
    // default:
    //   return <NoVisContent />;
  }
};

export const VisToolbar: NavDrawerItem = {
  key: "visualization",
  content: <Content />,
  icon: <Icon.Visualize />,
  minSize: 125,
  maxSize: 250,
};
