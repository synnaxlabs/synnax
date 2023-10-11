// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Align } from "@synnaxlabs/pluto";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { type Layout } from "@/layout";
import { Range } from "@/range";

const Content = (): ReactElement => {
  return (
    <Align.Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Workspace />}>Workspace</ToolbarTitle>
      </ToolbarHeader>
      <Range.List />
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "workspace",
  icon: <Icon.Workspace />,
  content: <Content />,
  tooltip: "Workspace",
  initialSize: 350,
  minSize: 250,
  maxSize: 500,
};
