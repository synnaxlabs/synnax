// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type batch,
  group,
  imex,
  lineplot,
  log,
  type ontology,
  panel,
  project,
  resource,
  schematic,
  type Synnax as Client,
  table,
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

type Visualization = "schematic" | "lineplot" | "log" | "table" | "panel";

interface VisualizationSpec {
  source: (client: Client, project: project.Key) => Tree.Source;
  inGroups: (client: Client, groups: group.Key[]) => Promise<string[]>;
  setProject: (keys: string[], to: project.Key) => batch.Request;
}

const keysOf = (records: Array<{ key: string }>): string[] =>
  records.map(({ key }) => key);

const VISUALIZATIONS: Record<Visualization, VisualizationSpec> = {
  schematic: {
    source: (client, key) =>
      Tree.createSource(
        client.schematics,
        { projects: [key], grouped: false },
        Tree.toResource("schematic"),
      ),
    inGroups: async (client, groups) =>
      keysOf(await client.schematics.retrieve({ groups })),
    setProject: schematic.setProjectRequest,
  },
  lineplot: {
    source: (client, key) =>
      Tree.createSource(
        client.lineplots,
        { projects: [key], grouped: false },
        Tree.toResource("lineplot"),
      ),
    inGroups: async (client, groups) =>
      keysOf(await client.lineplots.retrieve({ groups })),
    setProject: lineplot.setProjectRequest,
  },
  log: {
    source: (client, key) =>
      Tree.createSource(
        client.logs,
        { projects: [key], grouped: false },
        Tree.toResource("log"),
      ),
    inGroups: async (client, groups) => keysOf(await client.logs.retrieve({ groups })),
    setProject: log.setProjectRequest,
  },
  table: {
    source: (client, key) =>
      Tree.createSource(
        client.tables,
        { projects: [key], grouped: false },
        Tree.toResource("table"),
      ),
    inGroups: async (client, groups) =>
      keysOf(await client.tables.retrieve({ groups })),
    setProject: table.setProjectRequest,
  },
  panel: {
    source: (client, key) =>
      Tree.createSource(
        client.panels,
        { projects: [key], grouped: false },
        Tree.toResource("panel"),
      ),
    inGroups: async (client, groups) =>
      keysOf(await client.panels.retrieve({ groups })),
    setProject: panel.setProjectRequest,
  },
};

const VISUALIZATION_TYPES = Object.keys(VISUALIZATIONS) as Visualization[];

const isVisualization = (type: resource.Type): type is Visualization =>
  type in VISUALIZATIONS;

const DROPPABLE_TYPES = new Set<resource.Type>([...VISUALIZATION_TYPES, "group"]);

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
    { scopes: ["project"], nested: false },
    Tree.toResource("group"),
  ),
  Tree.createSource(client.projects, { grouped: false }, Tree.toResource("project")),
];

const children: Tree.Children = ({ client, id }) => [
  Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
  ...VISUALIZATION_TYPES.map((type) => VISUALIZATIONS[type].source(client, id.key)),
];

const descendantGroups = async (
  client: Client,
  roots: group.Key[],
): Promise<group.Key[]> => {
  const out = [...roots];
  let frontier = roots;
  while (frontier.length > 0) {
    const next = await client.groups.retrieve({ parents: group.resourceID(frontier) });
    frontier = next.map(({ key }) => key);
    out.push(...frontier);
  }
  return out;
};

// A visualization from another project changes project, and so does every
// visualization inside a group from another project.
const setProjectRequests = async (
  client: Client,
  items: Tree.Dropped[],
  destination: project.Key,
): Promise<batch.Request[]> => {
  const foreign = items.filter(({ ancestors }) => {
    const source = ancestors.find(
      (id) => id.type === "project" && !resource.idsEqual(id, project.TYPE_RESOURCE_ID),
    );
    return source != null && source.key !== destination;
  });
  const byType = new Map<Visualization, string[]>();
  const add = (type: Visualization, keys: string[]) =>
    byType.set(type, [...(byType.get(type) ?? []), ...keys]);
  foreign.forEach(({ id }) => {
    if (isVisualization(id.type)) add(id.type, [id.key]);
  });
  const groups = foreign
    .filter(({ id }) => id.type === "group")
    .map(({ id }) => id.key);
  if (groups.length > 0) {
    const nested = await descendantGroups(client, groups);
    await Promise.all(
      VISUALIZATION_TYPES.map(async (type) =>
        add(type, await VISUALIZATIONS[type].inGroups(client, nested)),
      ),
    );
  }
  return [...byType]
    .filter(([, keys]) => keys.length > 0)
    .map(([type, keys]) => VISUALIZATIONS[type].setProject(keys, destination));
};

const drop: Tree.Drop = async ({ client, destination, items }) => {
  await client.batch.send([
    ...Group.moveRequests(
      client,
      items.map(({ id }) => id),
      destination.id,
    ),
    ...(await setProjectRequests(client, items, destination.id.key)),
  ]);
};

const TreeItem = Tree.createItem({
  type: "project",
  icon: <Icon.Project />,
  useOnSelect,
  children,
  ContextMenu: TreeContextMenu,
  canDrop: ({ moved }) => moved.every(({ id }) => DROPPABLE_TYPES.has(id.type)),
  drop,
});

export const TREE_ITEMS = { project: TreeItem } satisfies Tree.Items;
