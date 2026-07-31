// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  lineplot,
  log,
  type ontology,
  project,
  schematic,
  table,
} from "@synnaxlabs/client";
import {
  Access,
  Icon,
  LinePlot as PLinePlot,
  Log as PLog,
  Menu,
  Project as Base,
  Schematic as PSchematic,
  Status,
  Synnax,
  Table as PTable,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { useExport } from "@/feature/project/export";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
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
  query: Base.useDelete,
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
  query: Base.useRename,
  ontologyID: project.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props): ReactElement => {
  const {
    selection,
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const group = Group.useCreateFromSelection();
  const projectKey = ids[0].key;
  const createPlot = LinePlot.useCreate({ project: projectKey });
  const createLog = Log.useCreate({ project: projectKey });
  const createTable = Table.useCreate({ project: projectKey });
  const firstID = selection.ids[0];
  const createSchematic = Schematic.useCreate({ project: projectKey });
  const importComponent = Import.useImport();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = useExport();
  const handleRename = useRename(props);
  const resources = getResource(ids);
  const first = resources[0];
  const singleResource = resources.length === 1;
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasLinePlotCreatePermission = Access.useCreateGranted(
    lineplot.TYPE_ONTOLOGY_ID,
  );
  const hasLogCreatePermission = Access.useCreateGranted(log.TYPE_ONTOLOGY_ID);
  const hasTableCreatePermission = Access.useCreateGranted(table.TYPE_ONTOLOGY_ID);
  const hasSchematicCreatePermission = Access.useCreateGranted(
    schematic.TYPE_ONTOLOGY_ID,
  );
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && singleResource && (
        <>
          <ContextMenu.RenameItem onClick={handleRename} />
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          shape={shape}
          rootID={rootID}
          onClick={() => group(props)}
        />
      )}
      {hasUpdatePermission || (hasDeletePermission && <Menu.Divider />)}
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
              Import component(s)
            </Menu.Item>
          )}
          <Menu.Divider />
          <Export.ContextMenuItem onClick={() => handleExport(first.id.key)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, ontologyID: first.id })}
          />
          <Tree.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      handleError(async () => {
        const proj = await client.projects.retrieve(resource.id.key);
        store.dispatch(Session.Project.select(proj.key));
      }, `Failed to select ${resource.name}`);
    },
    [client, store, handleError],
  );
};

const VALID_CHILDREN: ontology.ResourceType[] = [
  "schematic",
  "lineplot",
  "log",
  "table",
  "group",
];

const TreeItem = Tree.createItem({
  type: "project",
  icon: <Icon.Project />,
  useOnSelect,
  ContextMenu: TreeContextMenu,
  canDrop: ({ items }) =>
    items.every(({ key }) => VALID_CHILDREN.some((c) => key.toString().includes(c))),
});

export const TREE_ITEMS = { project: TreeItem } satisfies Tree.Items;
