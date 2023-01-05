// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Header, Space, Accordion } from "@synnaxlabs/pluto";
import type { NavDrawerItem } from "@synnaxlabs/pluto";
import { AiOutlinePlus } from "react-icons/ai";
import { MdWorkspacesFilled } from "react-icons/md";

import { LayoutList } from "./LayoutList";
import { RangesList } from "./RangesList";

import { Layout, useLayoutPlacer } from "@/features/layout";

const rangeWindowLayout: Layout = {
  key: "defineRange",
  type: "defineRange",
  title: "Define Range",
  location: "window",
  window: {
    resizable: false,
    height: 335,
    width: 550,
    navTop: true,
  },
};

const Content = (): JSX.Element => {
  const openWindow = useLayoutPlacer();
  return (
    <Space empty style={{ height: "100%" }}>
      <Header level="h4" divided>
        <Header.Title startIcon={<MdWorkspacesFilled />}>Workspace</Header.Title>
      </Header>
      <Accordion
        direction="vertical"
        entries={[
          {
            key: "ranges",
            title: "Ranges",
            content: <RangesList />,
            actions: [
              {
                children: <AiOutlinePlus />,
                onClick: () => openWindow(rangeWindowLayout),
              },
            ],
          },
          {
            key: "layouts",
            title: "Layouts",
            content: <LayoutList />,
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
