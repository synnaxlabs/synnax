// Copyright 2025 Synnax Labs, Inc.
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

import { Cluster } from "@/cluster";
import { Toolbar as Core } from "@/components";
import { type Layout } from "@/layout";
import { Tree } from "@/ontology/Tree";

const ResourcesTree = (): ReactElement => (
  <Align.Space empty style={{ height: "100%", position: "relative" }}>
    <Core.Header>
      <Core.Title icon={<Icon.Resources />}>Resources</Core.Title>
    </Core.Header>
    <Cluster.NoneConnectedBoundary>
      <Tree />
    </Cluster.NoneConnectedBoundary>
  </Align.Space>
);

export const Toolbar: Layout.NavDrawerItem = {
  key: "resources",
  icon: <Icon.Resources />,
  content: <ResourcesTree />,
  tooltip: "Resources",
  initialSize: 350,
  minSize: 150,
  maxSize: 400,
};
