// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group } from "@synnaxlabs/client";
import { Channel, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CREATE_LAYOUT } from "@/channel/Create";
import { Toolbar } from "@/components";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";

const Content = (): ReactElement => {
  const { data: g } = Channel.useRetrieveGroup({});
  const placeLayout = Layout.usePlacer();
  return (
    <Ontology.Toolbar>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Channel />}>Channels</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action onClick={() => placeLayout(CREATE_LAYOUT)}>
            <Icon.Add />
          </Toolbar.Action>
        </Toolbar.Actions>
      </Toolbar.Header>
      <Ontology.Tree root={g == null ? undefined : group.ontologyID(g.key)} />
    </Ontology.Toolbar>
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
