// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type ontology, workspace } from "@synnaxlabs/client";
import {
  ContextMenu as PContextMenu,
  Icon,
  LinePlot as PLinePlot,
  Log as PLog,
  Schematic as PSchematic,
  Synnax,
  Table as PTable,
  Workspace as Core,
} from "@synnaxlabs/pluto";
import { array, deep, strings } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Export } from "@/export";
import { EXTRACTORS } from "@/extractors";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { LinePlotServices } from "@/lineplot/services";
import { Link } from "@/link";
import { Log } from "@/log";
import { LogServices } from "@/log/services";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";
import { Schematic } from "@/schematic";
import { SchematicServices } from "@/schematic/services";
import { Table } from "@/table";
import { TableServices } from "@/table/services";
import { useExport } from "@/workspace/export";
import { selectActiveKey, useSelectActiveKey } from "@/workspace/selectors";
import { maybeRename, setActive } from "@/workspace/slice";

const useDelete = createUseDelete({
  type: "Workspace",
  query: Core.useDelete,
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
      const { key, name, snapshot } = schematic;
      placeLayout(Schematic.create({ ...schematic.data, key, name, snapshot }));
    },
  });
  return useCallback(
    () =>
      update({
        workspace: workspaceID.key,
        name: "New Schematic",
        snapshot: false,
        data: deep.copy(Schematic.ZERO_STATE),
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
  query: Core.useRename,
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
    selection: { ids },
    state: { getResource },
  } = props;
  const handleDelete = useDelete(props);
  const handleCreatePlot = useCreateLinePlot(props);
  const handleCreateLog = useCreateLog(props);
  const handleCreateTable = useCreateTable(props);
  const firstID = selection.ids[0];
  const handleImportPlot = LinePlotServices.useImport(firstID.key);
  const handleCreateSchematic = useCreateSchematic(props);
  const importSchematic = SchematicServices.useImport(firstID.key);
  const copyLinkToClipboard = Cluster.useCopyLinkToClipboard();
  const exportWS = useExport(EXTRACTORS);
  const handleImportLog = LogServices.useImport(firstID.key);
  const importTable = TableServices.useImport(firstID.key);
  const handleRename = useRename(props);
  const resources = getResource(ids);
  const first = resources[0];
  const handleExport = () => exportWS(first.id.key);
  const handleLink = () =>
    copyLinkToClipboard({ name: first.name, ontologyID: first.id });
  const singleResource = resources.length === 1;
  const canCreateSchematic = Schematic.useSelectHasPermission();
  return (
    <>
      {singleResource && (
        <ContextMenu.RenameItem onClick={handleRename} showBottomDivider />
      )}
      <ContextMenu.DeleteItem onClick={handleDelete} />
      <Group.ContextMenuItem {...props} showBottomDivider />
      {singleResource && (
        <>
          <PContextMenu.Item onClick={handleCreatePlot}>
            <LinePlotServices.CreateIcon />
            Create line plot
          </PContextMenu.Item>
          <PContextMenu.Item onClick={handleCreateLog}>
            <LogServices.CreateIcon />
            Create log
          </PContextMenu.Item>
          {canCreateSchematic && (
            <PContextMenu.Item onClick={handleCreateSchematic}>
              <SchematicServices.CreateIcon />
              Create schematic
            </PContextMenu.Item>
          )}
          <PContextMenu.Item onClick={handleCreateTable} showBottomDivider>
            <TableServices.CreateIcon />
            Create table
          </PContextMenu.Item>
          <PContextMenu.Item onClick={handleImportPlot}>
            <LinePlotServices.ImportIcon />
            Import line plot(s)
          </PContextMenu.Item>
          <PContextMenu.Item onClick={handleImportLog}>
            <LogServices.ImportIcon />
            Import log(s)
          </PContextMenu.Item>
          {canCreateSchematic && (
            <PContextMenu.Item onClick={importSchematic}>
              <SchematicServices.ImportIcon />
              Import schematic(s)
            </PContextMenu.Item>
          )}
          <PContextMenu.Item onClick={importTable} showBottomDivider>
            <TableServices.ImportIcon />
            Import table(s)
          </PContextMenu.Item>
          <Export.ContextMenuItem onClick={handleExport} />
          <Link.CopyContextMenuItem onClick={handleLink} />
          <Ontology.CopyContextMenuItem {...props} showBottomDivider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </>
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
