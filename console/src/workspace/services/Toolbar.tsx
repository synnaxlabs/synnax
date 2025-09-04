// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon, Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { EmptyAction, Toolbar } from "@/components";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { CREATE_LAYOUT } from "@/workspace/Create";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery({
    queryKey: [client?.key, "workspace-group"],
    queryFn: async () => {
      if (client == null) return null;
      const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
      return res.filter((r) => r.name === "Workspaces")[0].id;
    },
  });
  const placeLayout = Layout.usePlacer();

  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Workspace />}>Workspaces</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action onClick={() => placeLayout(CREATE_LAYOUT)}>
            <Icon.Add />
          </Toolbar.Action>
        </Toolbar.Actions>
      </Toolbar.Header>
      <Ontology.Tree root={group.data} emptyContent={<EmptyContent />} />
    </Toolbar.Content>
  );
};

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const handleClick = () => placeLayout(CREATE_LAYOUT);
  return (
    <EmptyAction
      message="No workspaces found."
      action="Create a workspace"
      onClick={handleClick}
    />
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
