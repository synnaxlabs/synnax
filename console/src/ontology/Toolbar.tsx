// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Toolbar } from "@/components";
import { type Layout } from "@/layout";
import { Tree } from "@/ontology/Tree";

const Content = (): ReactElement => (
  <Toolbar.Content>
    <Toolbar.Header padded>
      <Toolbar.Title icon={<Icon.Resources />}>Resources</Toolbar.Title>
    </Toolbar.Header>
    <Tree root={ontology.ROOT_ID} />
  </Toolbar.Content>
);

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
