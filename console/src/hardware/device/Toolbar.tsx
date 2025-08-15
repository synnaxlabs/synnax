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
import { Ontology } from "@/ontology";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery({
    queryKey: [client?.key, "device-group"],
    queryFn: async () => {
      if (client == null) return null;
      const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
      return res.filter((r) => r.name === "Devices")[0].id;
    },
  });
  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Device />}>Devices</Toolbar.Title>
      </Toolbar.Header>
      <Ontology.Tree root={group.data} />
    </Toolbar.Content>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "device",
  icon: <Icon.Device />,
  content: <Content />,
  tooltip: "Devices",
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
  trigger: ["D"],
};
