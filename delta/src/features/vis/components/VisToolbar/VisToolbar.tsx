// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Space, Text } from "@synnaxlabs/pluto";

import { useControlledVis } from "../../hooks";
import { LinePlotToolBar } from "../line/controls/LinePlotToolbar";
import { ControlledLineVisProps } from "../line/controls/types";
import { Vis } from "../../types";

import { VisIcon, VisToolbarTitle } from "./VisToolbarTitle";

import { ToolbarHeader } from "@/components";

const NoVisContent = (): JSX.Element => (
  <Space justify="spaceBetween" style={{ height: "100%" }} empty>
    <ToolbarHeader>
      <VisToolbarTitle />
    </ToolbarHeader>
    <Space.Centered>
      <Space direction="x" align="center" size="small">
        <Text level="h4" style={{ color: "var(--pluto-gray-m0)" }}>
          No Active Visualization. Select a Tab or
        </Text>
        <Text.Link level="h4" onClick={console.log}>
          Create a New One
        </Text.Link>
      </Space>
    </Space.Centered>
  </Space>
);

const Content = (): JSX.Element => {
  const controlled = useControlledVis<Vis>();
  if (controlled == null) return <NoVisContent />;

  switch (controlled.vis.variant) {
    default:
      return <LinePlotToolBar {...(controlled as unknown as ControlledLineVisProps)} />;
  }
};

export const VisToolbar = {
  key: "visualization",
  content: <Content />,
  icon: <VisIcon />,
  minSize: 125,
  maxSize: 250,
};
