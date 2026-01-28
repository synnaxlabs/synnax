// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type ontology, workspace } from "@synnaxlabs/client";
import {
  Icon,
  LinePlot as PLinePlot,
  Log as PLog,
  Menu as PMenu,
  Schematic as PSchematic,
  Synnax,
  Table as PTable,
  Workspace as Base,
} from "@synnaxlabs/pluto";
import { array, deep, strings } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
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
  const importPlot = LinePlotServices.useImport(firstID.key);
  const createSchematic = useCreateSchematic(props);
  const importSchematic = SchematicServices.useImport(firstID.key);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = useExport(EXTRACTORS);
  const importLog = LogServices.useImport(firstID.key);
  const importTable = TableServices.useImport(firstID.key);
  const handleRename = useRename(props);
  const resources = getResource(ids);
  const first = resources[0];
  const handleSelect = {
    delete: handleDelete,
    rename: handleRename,
    group: () => group(props),
    createLog,
    createPlot,
    createTable,
    importPlot,
    importLog,
    importTable,
    createSchematic,
    importSchematic,
    export: () => handleExport(first.id.key),
    link: () => handleLink({ name: first.name, ontologyID: first.id }),
  };
  const singleResource = resources.length === 1;
  return (
    <PMenu.Menu onChange={handleSelect} level="small" background={1} gap="small">
      {singleResource && (
        <>
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.DeleteItem />
      <Group.MenuItem ids={ids} shape={shape} rootID={rootID} />
      <PMenu.Divider />
      {singleResource && (
        <>
          <PMenu.Item itemKey="createPlot">
            <LinePlotServices.CreateIcon />
            Create line plot
          </PMenu.Item>
          <PMenu.Item itemKey="createLog">
            <LogServices.CreateIcon />
            Create log
          </PMenu.Item>
          <PMenu.Item itemKey="createTable">
            <TableServices.CreateIcon />
            Create table
          </PMenu.Item>
          <PMenu.Item itemKey="createSchematic">
            <SchematicServices.CreateIcon />
            Create schematic
          </PMenu.Item>
          <PMenu.Divider />
          <PMenu.Item itemKey="importPlot">
            <LinePlotServices.ImportIcon />
            Import line plot(s)
          </PMenu.Item>
          <PMenu.Item itemKey="importLog">
            <LogServices.ImportIcon />
            Import log(s)
          </PMenu.Item>
          <PMenu.Item itemKey="importSchematic">
            <SchematicServices.ImportIcon />
            Import schematic(s)
          </PMenu.Item>
          <PMenu.Item itemKey="importTable">
            <TableServices.ImportIcon />
            Import table(s)
          </PMenu.Item>
          <PMenu.Divider />
          <Export.MenuItem />
          <Link.CopyMenuItem />
          <Ontology.CopyMenuItem {...props} />
          <PMenu.Divider />
        </>
      )}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
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
