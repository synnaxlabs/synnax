// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { CREATE_LAYOUT } from "@/channel/Create";
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
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Channel />}>Channels</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action onClick={() => placeLayout(CREATE_LAYOUT)}>
            <Icon.Add />
          </Toolbar.Action>
        </Toolbar.Actions>
      </Toolbar.Header>
      <Ontology.Tree root={group.data?.ontologyID} />
    </Toolbar.Content>
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
