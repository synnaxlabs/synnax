// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Space, Accordion } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { removeRange, setActiveRange, useSelectRange, useSelectRanges } from "../store";

import { rangeWindowLayout } from "./DefineRange";
import { RangesList } from "./RangesList";
import { VisList } from "./VisList";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { NavDrawerItem, setNavdrawerVisible, useLayoutPlacer } from "@/layout";
import { VisToolbar } from "@/vis";
import { createLineVis } from "@/vis/line";

const Content = (): ReactElement => {
  const newLayout = useLayoutPlacer();
  const dispatch = useDispatch();
  const ranges = useSelectRanges();
  const selectedRange = useSelectRange();

  const handleAddOrEditRange = (key?: string): void => {
    newLayout({
      ...rangeWindowLayout,
      key: key ?? rangeWindowLayout.key,
    });
  };

  const handleRemoveRange = (key: string): void => {
    dispatch(removeRange(key));
  };

  const handleSelectRange = (key: string): void => {
    dispatch(setActiveRange(key));
  };

  const handleCreateVis = (): void => {
    newLayout(createLineVis({}));
    dispatch(setNavdrawerVisible({ key: VisToolbar.key, value: true }));
  };

  return (
    <Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Workspace />}>Workspace</ToolbarTitle>
      </ToolbarHeader>
      <Accordion
        data={[
          {
            key: "ranges",
            name: "Ranges",
            content: (
              <RangesList
                ranges={ranges}
                selectedRange={selectedRange}
                onRemove={handleRemoveRange}
                onSelect={handleSelectRange}
                onAddOrEdit={handleAddOrEditRange}
              />
            ),
            actions: [
              {
                children: <Icon.Add />,
                onClick: () => handleAddOrEditRange(),
                sharp: true,
              },
            ],
          },
          {
            key: "visualizations",
            name: "Visualizations",
            content: <VisList />,
            actions: [
              {
                children: <Icon.Add />,
                onClick: () => handleCreateVis(),
              },
            ],
          },
        ]}
      />
    </Space>
  );
};

export const WorkspaceToolbar: NavDrawerItem = {
  key: "workspace",
  icon: <Icon.Workspace />,
  content: <Content />,
  initialSize: 350,
  minSize: 250,
  maxSize: 500,
};
