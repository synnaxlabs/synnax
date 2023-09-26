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
import { Align, Synnax, Text } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";
import { setNavdrawerVisible } from "@/layout/slice";
import { Tree } from "@/ontology/Tree";

import { SERVICES } from "./services";

const ResourcesTree = (): ReactElement => {
  const dispatch = useDispatch();
  const placer = Layout.usePlacer();
  const client = Synnax.use();

  const handleCluster: Text.TextProps["onClick"] = (e) => {
    e.stopPropagation();
    placer(Cluster.connectWindowLayout);
    dispatch(setNavdrawerVisible({ key: Cluster.Toolbar.key, value: true }));
  };

  if (client == null)
    return (
      <Align.Space empty style={{ height: "100%", position: "relative" }}>
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Resources />}>Resources</ToolbarTitle>
        </ToolbarHeader>
        <Align.Center direction="y" style={{ height: "100%" }} size="small">
          <Text.Text level="p">No cluster connected.</Text.Text>
          <Text.Link level="p" onClick={handleCluster}>
            Connect a cluster
          </Text.Link>
        </Align.Center>
      </Align.Space>
    );

  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Resources />}>Resources</ToolbarTitle>
      </ToolbarHeader>
      <Tree services={SERVICES} />
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "resources",
  icon: <Icon.Resources />,
  content: <ResourcesTree />,
  tooltip: "Resources",
  initialSize: 350,
  minSize: 250,
  maxSize: 650,
};
