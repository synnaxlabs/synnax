// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";

import { EmptyAction, Toolbar } from "@/components";
import { Layout } from "@/layout";
import { createSelectorLayout } from "@/layouts/Selector";
import { Ontology } from "@/ontology";
import { Workspace } from "@/workspace";
import { CREATE_LAYOUT } from "@/workspace/Create";

const Content = (): ReactElement => {
  const ws = Workspace.useSelectActive();
  const placeLayout = Layout.usePlacer();
  const groupID = useMemo(() => workspace.ontologyID(ws?.key ?? ""), [ws?.key]);
  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Workspace />}>Workspace</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action onClick={() => placeLayout(CREATE_LAYOUT)}>
            <Icon.Add />
          </Toolbar.Action>
        </Toolbar.Actions>
      </Toolbar.Header>
      <Ontology.Tree root={groupID} emptyContent={<EmptyContent />} />
    </Toolbar.Content>
  );
};

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  return (
    <EmptyAction
      message="No components in workspace."
      action="Create a component"
      onClick={() => placeLayout(createSelectorLayout({}))}
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
