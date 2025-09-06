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
import { type Layout } from "@/layout";
import { Tree } from "@/ontology/Tree";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery({
    queryKey: [client?.key, "user-group"],
    queryFn: async () => {
      if (client == null) return null;
      const { id } = await client.ontology.retrieve(ontology.ROOT_ID);
      return id;
    },
  });
  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Resources />}>Resources</Toolbar.Title>
      </Toolbar.Header>
      <Tree root={group.data} />
    </Toolbar.Content>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "ontology",
  icon: <Icon.Group />,
  content: <Content />,
  tooltip: "Resources",
  initialSize: 400,
  minSize: 175,
  maxSize: 400,
  trigger: ["O"],
};
