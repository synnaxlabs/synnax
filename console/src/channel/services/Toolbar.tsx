// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Icon, Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { CREATE_LAYOUT } from "@/channel/Create";
import { Cluster } from "@/cluster";
import { Toolbar } from "@/components";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery({
    queryKey: [client?.key, "channel-group"],
    queryFn: async () => {
      if (client == null) return null;
      return await client.channels.retrieveGroup();
    },
  });
  const placeLayout = Layout.usePlacer();
  return (
    <Cluster.NoneConnectedBoundary>
      <Align.Space empty style={{ height: "100%", overflow: "hidden" }}>
        <Toolbar.Header>
          <Toolbar.Title icon={<Icon.Channel />}>Channels</Toolbar.Title>
          <Toolbar.Actions>
            {[{ children: <Icon.Add />, onClick: () => placeLayout(CREATE_LAYOUT) }]}
          </Toolbar.Actions>
        </Toolbar.Header>

        <Ontology.Tree root={group.data?.ontologyID} />
      </Align.Space>
    </Cluster.NoneConnectedBoundary>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "channel",
  icon: <Icon.Channel />,
  content: <Content />,
  tooltip: "Channels",
  trigger: ["C"],
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};
