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

import { Cluster } from "@/cluster";
import { ToolbarHeader, ToolbarTitle } from "@/components";
import { type Layout } from "@/layout";
import { Tree } from "@/ontology/Tree";

const ResourcesTree = (): ReactElement => {
  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Resources />}>Resources</ToolbarTitle>
      </ToolbarHeader>
      <Cluster.NoneConnectedBoundary>
        <Tree />
      </Cluster.NoneConnectedBoundary>
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "resources",
  icon: <Icon.Resources />,
  content: <ResourcesTree />,
  tooltip: "Resources",
  initialSize: 350,
  minSize: 150,
  maxSize: 400,
};
