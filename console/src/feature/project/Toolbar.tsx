// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import { Access, Icon, Project } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Ontology } from "@/platform/ontology";
import { Project as CProject } from "@/platform/project";
import { Toolbar } from "@/platform/toolbar";

const Actions = (): ReactElement | null => {
  const openCreate = CProject.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  if (!hasCreatePermission) return null;
  return (
    <Toolbar.Actions>
      <Toolbar.Action onClick={() => openCreate()} tooltip="Create project">
        <Icon.Add />
      </Toolbar.Action>
    </Toolbar.Actions>
  );
};

const Content = (): ReactElement => {
  const { data: groupID } = Project.useRetrieveGroupID({});
  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Project />}>Projects</Toolbar.Title>
        <Actions />
      </Toolbar.Header>
      <Ontology.Tree root={groupID} emptyContent={<EmptyContent />} />
    </Toolbar.Content>
  );
};

const EmptyContent = () => {
  const openCreate = CProject.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  const handleClick = () => openCreate();
  return (
    <Empty.Action
      message="No projects found."
      action={hasCreatePermission ? "Create a project" : undefined}
      onClick={handleClick}
    />
  );
};

export const TOOLBAR: Nav.Item = {
  key: "project",
  icon: <Icon.Project />,
  content: <Content />,
  tooltip: "Projects",
  trigger: ["W"],
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
};
