// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { Cluster } from "@/cluster";
import { Toolbar } from "@/components";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { CREATE_LAYOUT } from "@/workspace/Create";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery<ontology.ID | null>({
    queryKey: [client?.key, "workspace-group"],
    queryFn: async () => {
      if (client == null) return null;
      const res = await client.ontology.retrieveChildren(ontology.ROOT_ID, {
        includeSchema: false,
      });
      return res.find(({ name }) => name === "Workspaces")?.id ?? null;
    },
  });
  const placeLayout = Layout.usePlacer();

  return (
    <Cluster.NoneConnectedBoundary>
      <Align.Space empty style={{ height: "100%" }}>
        <Toolbar.Header>
          <Toolbar.Title icon={<Icon.Workspace />}>Workspaces</Toolbar.Title>
          <Toolbar.Actions>
            {[
              {
                key: "create",
                children: <Icon.Add />,
                onClick: () => placeLayout(CREATE_LAYOUT),
              },
            ]}
          </Toolbar.Actions>
        </Toolbar.Header>
        <Ontology.Tree root={group.data ?? undefined} />
      </Align.Space>
    </Cluster.NoneConnectedBoundary>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "workspace",
  icon: <Icon.Workspace />,
  content: <Content />,
  tooltip: "Workspaces",
  trigger: ["W"],
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};
