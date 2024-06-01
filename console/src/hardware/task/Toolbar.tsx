// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";

const Content = (): ReactElement => (
  <Align.Space empty style={{ height: "100%" }}>
    <ToolbarHeader>
      <ToolbarTitle icon={<Icon.Task />}>Tasks</ToolbarTitle>
    </ToolbarHeader>
  </Align.Space>
);

export const Toolbar: Layout.NavDrawerItem = {
  key: "task",
  icon: <Icon.Task />,
  content: <Content />,
  tooltip: "Tasks",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};
