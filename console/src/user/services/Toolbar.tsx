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

import { Toolbar } from "@/components";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { REGISTER_LAYOUT } from "@/user/Register";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery({
    queryKey: [client?.key, "user-group"],
    queryFn: async () => {
      if (client == null) return null;
      const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
      return res.filter((r) => r.name === "Users")[0].id;
    },
  });
  const placeLayout = Layout.usePlacer();

  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.User />}>Users</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action onClick={() => placeLayout(REGISTER_LAYOUT)}>
            <Icon.Add />
          </Toolbar.Action>
        </Toolbar.Actions>
      </Toolbar.Header>
      <Ontology.Tree root={group.data} />
    </Toolbar.Content>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "user",
  icon: <Icon.User />,
  content: <Content />,
  tooltip: "Users",
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
  trigger: ["U"],
};
