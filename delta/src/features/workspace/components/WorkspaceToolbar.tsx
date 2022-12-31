// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Header, Space, Accordion } from "@synnaxlabs/pluto";
import { AiOutlinePlus } from "react-icons/ai";
import { MdWorkspacesFilled } from "react-icons/md";

import { RangesAccordionEntry } from "./RangesList";

import { Layout, useLayoutPlacer } from "@/features/layout";

const rangeWindowLayout: Layout = {
  key: "defineRange",
  type: "defineRange",
  title: "Define Range",
  location: "window",
  window: {
    resizable: false,
    height: 330,
    width: 550,
    navTop: true,
  },
};

const Content = (): JSX.Element => {
  const openWindow = useLayoutPlacer();
  return (
    <Space empty style={{ height: "100%" }}>
      <Header level="h4" divided icon={<MdWorkspacesFilled />}>
        Workspace
      </Header>
      <Accordion
        direction="vertical"
        entries={[
          {
            key: "ranges",
            title: "Ranges",
            content: <RangesAccordionEntry />,
            actions: [
              {
                children: <AiOutlinePlus />,
                onClick: () => openWindow(rangeWindowLayout),
              },
            ],
          },
        ]}
      />
    </Space>
  );
};

export const WorkspaceToolbar = {
  key: "workspace",
  icon: <MdWorkspacesFilled />,
  content: <Content />,
  initialSize: 350,
};
