// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  imex,
  lineplot,
  log,
  project,
  type resource,
  schematic,
  type Synnax as Client,
  table,
  UnexpectedError,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import {
  Access,
  LinePlot as PLinePlot,
  Log as PLog,
  Project as Base,
  Schematic as PSchematic,
  Synnax,
  Table as PTable,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Import } from "@/platform/import";
import { LinePlot } from "@/platform/lineplot";
import { Link } from "@/platform/link";
import { Log } from "@/platform/log";
import { Schematic } from "@/platform/schematic";
import { Table } from "@/platform/table";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useDelete = Tree.createUseDelete({
  type: "Project",
  useUpdate: Base.useDelete,
  convertKey: String,
  afterSuccess: ({ data, store }) => {
    const s = store.getState();
    const activeKey = Session.Project.selectOptionalSelected(s);
    const active = array.toArray(data).find((k) => k === activeKey);
    if (active == null) return;
    store.dispatch(Session.Project.clearSelected());
    store.dispatch(Session.Panel.reset());
  },
});

const useRename = Tree.createUseRename({
  useUpdate: Base.useRename,
  resourceID: project.resourceID,
  convertKey: String,
});

const retrieve: Tree.Retrieve = async (c, key) => await c.projects.retrieve({ key });

const TreeContextMenu: Tree.ContextMenu = (props): ReactElement => {
  const {
    selection,
    selection: { ids, rootID, parentID },
    state,
  } = props;
  const handleDelete = useDelete(props);
  const group = Group.useCreateFromSelection();
  const projectKey = ids[0].key;
  const createPlot = LinePlot.useCreate({ project: projectKey });
  const createLog = Log.useCreate({ project: projectKey });
  const createTable = Table.useCreate({ project: projectKey });
  const firstID = selection.ids[0];
  const createSchematic = Schematic.useCreate({ project: projectKey });
  const importComponent = Import.use();
  const handleLink = Core.useCopyLinkToClipboard();
  const handleExport = Export.use();
  const handleRename = useRename(props);
  const resources = Tree.useResources(state, ids);
  const first = resources[0];
  const singleResource = resources.length === 1;
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasLinePlotCreatePermission = Access.useCreateGranted(
    lineplot.TYPE_RESOURCE_ID,
  );
  const hasLogCreatePermission = Access.useCreateGranted(log.TYPE_RESOURCE_ID);
  const hasTableCreatePermission = Access.useCreateGranted(table.TYPE_RESOURCE_ID);
  const hasSchematicCreatePermission = Access.useCreateGranted(
    schematic.TYPE_RESOURCE_ID,
  );
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          {singleResource && <ContextMenu.RenameItem onClick={handleRename} />}
          <Group.ContextMenuItem
            ids={ids}
            shape={state.shape}
            rootID={rootID}
            parentID={parentID}
            onClick={() => group(props)}
          />
        </>
      )}
      <Menu.Divider />
      {singleResource && (
        <>
          {hasLinePlotCreatePermission && (
            <Menu.Item itemKey="createPlot" onClick={() => createPlot()}>
              <PLinePlot.CreateIcon />
              Create line plot
            </Menu.Item>
          )}
          {hasLogCreatePermission && (
            <Menu.Item itemKey="createLog" onClick={() => createLog()}>
              <PLog.CreateIcon />
              Create log
            </Menu.Item>
          )}
          {hasTableCreatePermission && (
            <Menu.Item itemKey="createTable" onClick={() => createTable()}>
              <PTable.CreateIcon />
              Create table
            </Menu.Item>
          )}
          {hasSchematicCreatePermission && (
            <Menu.Item itemKey="createSchematic" onClick={() => createSchematic()}>
              <PSchematic.CreateIcon />
              Create schematic
            </Menu.Item>
          )}
          <Menu.Divider />
          {hasUpdatePermission && (
            <Menu.Item itemKey="import" onClick={() => importComponent(firstID.key)}>
              <Icon.Import />
              Import components
            </Menu.Item>
          )}
        </>
      )}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      {singleResource && (
        <>
          <Export.ContextMenuItem
            onClick={() =>
              handleExport({
                stream: (client) =>
                  client.projects.export(first.id.key, imex.JSON_OPTIONS),
                name: first.name,
                extension: "zip",
              })
            }
          />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, resourceID: first.id })}
          />
          <Tree.CopyPropertiesContextMenuItem {...props} retrieve={retrieve} />
        </>
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const useOnSelect = (): ((resource: Tree.Resource) => void) => {
  const client = Synnax.use();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  return (resource) => {
    if (client == null) return;
    handleError(
      async () => {
        const proj = await client.projects.retrieve(resource.id.key);
        store.dispatch(Session.Project.select(proj.key));
      },
      { message: "Failed to select project", name: resource.name },
    );
  };
};

const DROPPABLE_TYPES = new Set<resource.Type>([...Base.VISUALIZATION_TYPES, "group"]);

/** Lists the projects that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "project",
  children: (client, key) =>
    Tree.createSource(client.projects, { groups: [key] }, Tree.toResource("project")),
};

/** Lists the top level of the project tree: its groups and the projects in no group. */
export const listTreeRoot = (client: Client): Tree.Source[] => [
  Tree.createSource(
    client.groups,
    { scope: "project", nested: false },
    Tree.toResource("group"),
  ),
  Tree.createSource(client.projects, { grouped: false }, Tree.toResource("project")),
];

const children: Tree.Children = ({ client, id }) => [
  Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
  ...Base.VISUALIZATION_TYPES.map((type) =>
    Tree.createSource(
      Base.VISUALIZATIONS[type](client),
      { projects: [id.key], grouped: false },
      Tree.toResource(type),
    ),
  ),
];

// A dropped visualization leaves its group. The Core moves the visualizations in a
// dropped group along with it.
const drop: Tree.Drop = async ({ client, destination, items }) => {
  const groups: resource.ID[] = [];
  const byType = new Map<Base.VisualizationType, string[]>();
  items.forEach(({ id }) => {
    if (id.type === "group") groups.push(id);
    else if (Base.isVisualizationType(id.type))
      byType.set(id.type, [...(byType.get(id.type) ?? []), id.key]);
    else throw new UnexpectedError(`a ${id.type} does not belong to a project`);
  });
  await client.batch.send([
    ...client.groups.moveRequests(groups, destination.id),
    ...[...byType].flatMap(([type, keys]) =>
      Base.VISUALIZATIONS[type](client).setProjectRequest(keys, destination.id.key),
    ),
  ]);
};

const TreeItem = Tree.createItem({
  linkable: true,
  icon: <Icon.Project />,
  useOnSelect,
  children,
  ContextMenu: TreeContextMenu,
  canDrop: ({ moved }) => moved.every(({ id }) => DROPPABLE_TYPES.has(id.type)),
  drop,
});

export const TREE_ITEMS = { project: TreeItem } satisfies Tree.Items;
