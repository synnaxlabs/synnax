// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  lineplot,
  log,
  type ontology,
  schematic,
  table,
  workspace,
} from "@synnaxlabs/client";
import {
  Access,
  Icon,
  LinePlot as PLinePlot,
  Log as PLog,
  Menu,
  Schematic as PSchematic,
  Synnax,
  Table as PTable,
  Workspace as Base,
} from "@synnaxlabs/pluto";
import { array, deep, strings } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Import } from "@/import";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Log } from "@/log";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";
import { Schematic } from "@/schematic";
import { Table } from "@/table";
import { useExport } from "@/workspace/export";
import { selectActiveKey, useSelectActiveKey } from "@/workspace/selectors";
import { maybeRename, setActive } from "@/workspace/slice";

const useDelete = createUseDelete({
  type: "Workspace",
  query: Base.useDelete,
  convertKey: String,
  afterSuccess: ({ data, store }) => {
    const s = store.getState();
    const activeKey = selectActiveKey(s);
    const active = array.toArray(data).find((k) => k === activeKey);
    if (active == null) return;
    store.dispatch(setActive(null));
    store.dispatch(Layout.clearWorkspace());
  },
});

const useMaybeChangeWorkspace = (): ((key: string) => Promise<void>) => {
  const dispatch = useDispatch();
  const activeWS = useSelectActiveKey();
  const client = Synnax.use();
  return async (key) => {
    if (activeWS === key) return;
    if (client == null) throw new DisconnectedError();
    const { layout, ...ws } = await client.workspaces.retrieve(key);
    dispatch(setActive(ws));
    dispatch(
      Layout.setWorkspace({ slice: layout as Layout.SliceState, keepNav: false }),
    );
  };
};

const useCreateSchematic = ({
  placeLayout,
  selection: { ids },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const maybeChangeWorkspace = useMaybeChangeWorkspace();
  const workspaceID = ids[0];
  const { update } = PSchematic.useCreate({
    afterSuccess: async ({ data }) => {
      const { workspace, ...schematic } = data;
      await maybeChangeWorkspace(workspace);
      placeLayout(Schematic.create({ ...schematic }));
    },
  });
  return useCallback(
    () =>
      update({
        workspace: workspaceID.key,
        name: "New Schematic",
        snapshot: false,
        authority: 1,
        legend: deep.copy(Schematic.ZERO_STATE.legend),
        nodes: [],
        edges: [],
        props: {},
      }),
    [workspaceID.key],
  );
};

const useCreateLinePlot = ({
  placeLayout,
  selection: { ids },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const maybeChangeWorkspace = useMaybeChangeWorkspace();
  const workspaceID = ids[0];
  const { update } = PLinePlot.useCreate({
    afterSuccess: async ({ data }) => {
      const { workspace, ...linePlot } = data;
      await maybeChangeWorkspace(workspaceID.key);
      placeLayout(LinePlot.create({ ...linePlot.data, ...linePlot }));
    },
  });
  return useCallback(
    () =>
      update({
        workspace: workspaceID.key,
        name: "New Line Plot",
        data: deep.copy(LinePlot.ZERO_SLICE_STATE),
      }),
    [workspaceID.key],
  );
};

const useCreateLog = ({
  placeLayout,
  selection: { ids },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const maybeChangeWorkspace = useMaybeChangeWorkspace();
  const workspaceID = ids[0];
  const { update } = PLog.useCreate({
    afterSuccess: async ({ data }) => {
      const { workspace, ...log } = data;
      await maybeChangeWorkspace(workspace);
      placeLayout(Log.create({ ...log.data, key: log.key, name: log.name }));
    },
  });
  return useCallback(
    () =>
      update({
        workspace: workspaceID.key,
        name: "New Log",
        data: deep.copy(Log.ZERO_STATE),
      }),
    [workspaceID.key],
  );
};

const useCreateTable = ({
  placeLayout,
  selection: { ids },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const maybeChangeWorkspace = useMaybeChangeWorkspace();
  const workspaceID = ids[0];
  const { update } = PTable.useCreate({
    afterSuccess: async ({ data }) => {
      const { workspace, ...table } = data;
      await maybeChangeWorkspace(workspace);
      placeLayout(Table.create({ ...table.data, key: table.key, name: table.name }));
    },
  });
  return useCallback(
    () =>
      update({
        workspace: workspaceID.key,
        name: "New Table",
        data: deep.copy(Table.ZERO_STATE),
      }),
    [workspaceID.key],
  );
};

const useRename = createUseRename({
  query: Base.useRename,
  ontologyID: workspace.ontologyID,
  convertKey: String,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    store.dispatch(maybeRename({ key, name }));
    rollbacks.push(() => store.dispatch(maybeRename({ key, name: oldName })));
    return { ...data, name };
  },
});

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    selection,
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const group = Group.useCreateFromSelection();
  const createPlot = useCreateLinePlot(props);
  const createLog = useCreateLog(props);
  const createTable = useCreateTable(props);
  const firstID = selection.ids[0];
  const createSchematic = useCreateSchematic(props);
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
            <Menu.Item itemKey="createPlot" onClick={createPlot}>
              <PLinePlot.CreateIcon />
              Create line plot
            </Menu.Item>
          )}
          {hasLogCreatePermission && (
            <Menu.Item itemKey="createLog" onClick={createLog}>
              <PLog.CreateIcon />
              Create log
            </Menu.Item>
          )}
          {hasTableCreatePermission && (
            <Menu.Item itemKey="createTable" onClick={createTable}>
              <PTable.CreateIcon />
              Create table
            </Menu.Item>
          )}
          {hasSchematicCreatePermission && (
            <Menu.Item itemKey="createSchematic" onClick={createSchematic}>
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
          <Ontology.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const handleSelect: Ontology.HandleSelect = ({
  selection,
  client,
  store,
  handleError,
}) => {
  const names = strings.naturalLanguageJoin(
    selection.map(({ name }) => name),
    "workspace",
  );
  handleError(async () => {
    const ws = await client.workspaces.retrieve(selection[0].id.key);
    store.dispatch(setActive(ws));
    store.dispatch(
      Layout.setWorkspace({ slice: ws.layout as Layout.SliceState, keepNav: false }),
    );
  }, `Failed to select ${names}`);
};

const VALID_CHILDREN: ontology.ResourceType[] = [
  "schematic",
  "lineplot",
  "log",
  "table",
  "group",
];

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "workspace",
  icon: <Icon.Workspace />,
  onSelect: handleSelect,
  TreeContextMenu,
  canDrop: ({ items }) =>
    items.every(({ key }) => VALID_CHILDREN.some((c) => key.toString().includes(c))),
};
