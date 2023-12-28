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
import { Align, Header } from "@synnaxlabs/pluto";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";
import { Range } from "@/range";
import { defineWindowLayout } from "@/range/Define";

const Content = (): ReactElement => {
  const p = Layout.usePlacer();
  return (
    <Align.Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Range />}>Ranges</ToolbarTitle>
        <Header.Actions>
          {[
            {
              children: <Icon.Add />,
              onClick: () => p(defineWindowLayout),
            },
          ]}
        </Header.Actions>
      </ToolbarHeader>
      <Range.List />
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "range",
  icon: <Icon.Range />,
  content: <Content />,
  tooltip: "Workspace",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};
