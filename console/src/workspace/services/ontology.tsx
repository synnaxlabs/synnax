// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, ontology } from "@synnaxlabs/client";
import {
  Icon,
  LinePlot as PLinePlot,
  Log as PLog,
  Menu as PMenu,
  Schematic as PSchematic,
  Synnax,
  Table as PTable,
  Text,
  Workspace as Core,
} from "@synnaxlabs/pluto";
import { deep, type record, strings } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch, useStore } from "react-redux";

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
import { useConfirmDelete } from "@/ontology/hooks";
import { Schematic } from "@/schematic";
import { SchematicServices } from "@/schematic/services";
import { type RootState } from "@/store";
import { Table } from "@/table";
import { TableServices } from "@/table/services";
import { useExport } from "@/workspace/export";
import { select, selectActiveKey, useSelectActiveKey } from "@/workspace/selectors";
import { add, rename, setActive } from "@/workspace/slice";

const useDelete = ({
  selection: { ids },
  state: { getResource },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const confirm = useConfirmDelete({ type: "Workspace" });
  const keys = useMemo(() => ids.map((id) => id.key), [ids]);
  const store = useStore<RootState>();
  const dispatch = useDispatch();
  const beforeUpdate = useCallback(async () => await confirm(getResource(ids)), [ids]);
  const afterUpdate = useCallback(() => {
    const s = store.getState();
    const activeKey = selectActiveKey(s);
    const active = ids.find((id) => id.key === activeKey);
    if (active != null) {
      dispatch(setActive(null));
      dispatch(Layout.clearWorkspace());
    }
  }, [ids, dispatch, store]);
  const { update } = Core.useDelete({ beforeUpdate, afterUpdate });
  return useCallback(() => update(keys), [keys]);
};

const useMaybeChangeWorkspace = (): ((key: string) => Promise<void>) => {
  const dispatch = useDispatch();
  const store = useStore<RootState>();
  const activeWS = useSelectActiveKey();
  const client = Synnax.use();
  return async (key) => {
    if (activeWS === key) return;
    let ws = select(store.getState(), key);
    if (ws == null) {
      if (client == null) throw new DisconnectedError();
      ws = await client.workspaces.retrieve(key);
    }
    dispatch(add(ws));
    dispatch(
      Layout.setWorkspace({ slice: ws.layout as Layout.SliceState, keepNav: false }),
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
    afterUpdate: async ({ value }) => {
      const { workspace, ...schematic } = value;
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
        data: deep.copy(Schematic.ZERO_STATE) as unknown as record.Unknown,
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
    afterUpdate: async ({ value }) => {
      const { workspace, ...linePlot } = value;
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
    afterUpdate: async ({ value }) => {
      const { workspace, ...log } = value;
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
    afterUpdate: async ({ value }) => {
      const { workspace, ...table } = value;
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
  const resources = getResource(ids);
  const first = resources[0];
  const handleSelect = {
    delete: handleDelete,
    rename: () => Text.edit(ontology.idToString(first.id)),
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
  const canCreateSchematic = Schematic.useSelectHasPermission();
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
            Create Line Plot
          </PMenu.Item>
          <PMenu.Item itemKey="createLog">
            <LogServices.CreateIcon />
            Create Log
          </PMenu.Item>
          <PMenu.Item itemKey="createTable">
            <TableServices.CreateIcon />
            Create Table
          </PMenu.Item>
          {canCreateSchematic && (
            <PMenu.Item itemKey="createSchematic">
              <SchematicServices.CreateIcon />
              Create Schematic
            </PMenu.Item>
          )}
          <PMenu.Divider />
          <PMenu.Item itemKey="importPlot">
            <LinePlotServices.ImportIcon />
            Import Line Plot(s)
          </PMenu.Item>
          <PMenu.Item itemKey="importLog">
            <LogServices.ImportIcon />
            Import Log(s)
          </PMenu.Item>
          {canCreateSchematic && (
            <PMenu.Item itemKey="importSchematic">
              <SchematicServices.ImportIcon />
              Import Schematic(s)
            </PMenu.Item>
          )}
          <PMenu.Item itemKey="importTable">
            <TableServices.ImportIcon />
            Import Table(s)
          </PMenu.Item>
          <PMenu.Divider />
          <Export.MenuItem />
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleSelect: Ontology.HandleSelect = ({
  selection,
  client,
  store,
  handleError,
}) => {
  client.workspaces
    .retrieve(selection[0].id.key)
    .then((workspace) => {
      store.dispatch(add(workspace));
      store.dispatch(
        Layout.setWorkspace({
          slice: workspace.layout as Layout.SliceState,
          keepNav: false,
        }),
      );
    })
    .catch((e) => {
      const names = strings.naturalLanguageJoin(
        selection.map(({ name }) => name),
        "workspace",
      );
      handleError(e, `Failed to select ${names}`);
    });
};

const handleRename: Ontology.HandleTreeRename = {
  eager: ({ id, name, store }) => store.dispatch(rename({ key: id.key, name })),
  execute: async ({ client, id, name }) => await client.workspaces.rename(id.key, name),
  rollback: ({ id, store }, prevName) =>
    store.dispatch(rename({ key: id.key, name: prevName })),
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
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
  canDrop: ({ items }) =>
    items.every(({ key }) => VALID_CHILDREN.some((c) => key.toString().includes(c))),
};
