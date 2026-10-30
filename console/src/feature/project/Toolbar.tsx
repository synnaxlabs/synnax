// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Ontology } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Project as PlatformProject } from "@/platform/project";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

const Content = (): ReactElement => {
  const openCreate = PlatformProject.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  const [searching, setSearching] = useState(false);
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title>
          <Icon.Project />
          Projects
        </Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.SearchAction value={searching} onChange={setSearching} />
          {hasCreatePermission && (
            <Toolbar.Action
              onClick={() => openCreate()}
              tooltip="Create project"
              variant="filled"
            >
              <Icon.Add />
            </Toolbar.Action>
          )}
        </Toolbar.Actions>
      </Toolbar.Header>
      <Toolbar.Body>
        <Body search={searching} />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

interface BodyProps {
  search: boolean;
}

const Body = ({ search }: BodyProps): ReactElement => {
  const groupID = Ontology.useRootChildID({ name: "Projects" });
  return <Tree.Tree root={groupID} search={search} emptyContent={<EmptyContent />} />;
};

const EmptyContent = () => {
  const openCreate = PlatformProject.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  const handleClick = () => openCreate();
  return (
    <Empty.Action
      message="No projects found"
      action={hasCreatePermission ? "Create project" : undefined}
      onClick={handleClick}
    />
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "project",
  icon: <Icon.Project />,
  content: <Content />,
  tooltip: "Projects",
  trigger: ["W"],
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
};
