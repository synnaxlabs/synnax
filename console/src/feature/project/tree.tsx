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
  type ontology,
  project,
  resource,
  schematic,
  table,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
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

const TreeContextMenu: Tree.ContextMenu = (props): ReactElement => {
  const {
    selection,
    selection: { ids, rootID },
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
          <Menu.Divider />
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
          <Tree.CopyPropertiesContextMenuItem {...props} />
        </>
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
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

const VISUALIZATION_TYPES: resource.Type[] = [
  "schematic",
  "lineplot",
  "log",
  "table",
  "panel",
];

const DROPPABLE_TYPES = new Set<resource.Type>([...VISUALIZATION_TYPES, "group"]);

/** Lists the projects that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "project",
  children: (client, key) =>
    Tree.createSource(client.projects, { groups: [key] }, Tree.toResource("project")),
};

const children: Tree.Children = ({ client, id }) => {
  if (resource.idsEqual(id, project.TYPE_RESOURCE_ID))
    return [
      Tree.createSource(
        client.groups,
        { scopes: ["project"], nested: false },
        Tree.toResource("group"),
      ),
      Tree.createSource(
        client.projects,
        { grouped: false },
        Tree.toResource("project"),
      ),
    ];
  return [
    Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
    // A project lists its visualizations through the ontology until PR 6.
    Tree.createOntologySource(
      client,
      id,
      VISUALIZATION_TYPES,
      ({ data }) => data?.group == null,
    ),
  ];
};

const drop: Tree.Drop = async ({ client, destination, items }) => {
  await client.batch.send(
    Group.moveRequests(
      client,
      items.map(({ id }) => id),
      destination.id,
    ),
  );
  // A visualization from another project moves its ontology edge until PR 6.
  const bySource = new Map<string, resource.ID[]>();
  items.forEach(({ id, ancestors }) => {
    if (id.type === "group") return;
    const source = ancestors.find(({ type }) => type === "project");
    if (source == null || resource.idsEqual(source, destination.id)) return;
    const key = resource.idToString(source);
    bySource.set(key, [...(bySource.get(key) ?? []), id]);
  });
  await Promise.all(
    [...bySource].map(
      async ([source, ids]) =>
        await client.ontology.moveChildren(
          resource.parseID(source),
          destination.id,
          ...ids,
        ),
    ),
  );
};

const TreeItem = Tree.createItem({
  type: "project",
  icon: <Icon.Project />,
  useOnSelect,
  children,
  ContextMenu: TreeContextMenu,
  canDrop: ({ items }) =>
    PTree.filterHaulItems(items).every(({ key }) =>
      DROPPABLE_TYPES.has(resource.parseID(key).type),
    ),
  drop,
});

export const TREE_ITEMS = { project: TreeItem } satisfies Tree.Items;
