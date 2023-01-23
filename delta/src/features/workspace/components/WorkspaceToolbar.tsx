// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Space, Accordion } from "@synnaxlabs/pluto";
import type { NavDrawerItem } from "@synnaxlabs/pluto";
import { AiOutlinePlus } from "react-icons/ai";
import { MdWorkspacesFilled } from "react-icons/md";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout, useLayoutPlacer } from "@/features/layout";

import { RangesList } from "./RangesList";
import { VisList } from "./VisList";

const rangeWindowLayout: Layout = {
  key: "defineRange",
  type: "defineRange",
  name: "Define Range",
  location: "window",
  window: {
    resizable: false,
    height: 340,
    width: 550,
    navTop: true,
  },
};

const Content = (): JSX.Element => {
  const openWindow = useLayoutPlacer();
  return (
    <Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<MdWorkspacesFilled />}>Workspace</ToolbarTitle>
      </ToolbarHeader>
      <Accordion
        data={[
          {
            key: "ranges",
            name: "Ranges",
            content: <RangesList />,
            actions: [
              {
                children: <AiOutlinePlus />,
                onClick: () => openWindow(rangeWindowLayout),
              },
            ],
          },
          {
            key: "visualizations",
            name: "Visualizations",
            content: <VisList />,
          },
        ]}
      />
    </Space>
  );
};

export const WorkspaceToolbar: NavDrawerItem = {
  key: "workspace",
  icon: <MdWorkspacesFilled />,
  content: <Content />,
  initialSize: 350,
  minSize: 250,
  maxSize: 500,
};
