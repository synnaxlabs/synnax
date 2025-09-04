// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  linePlot as clientLinePlot,
  log as clientLog,
  ontology,
  schematic as clientSchematic,
  table as clientTable,
} from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Synnax, Text, Tree } from "@synnaxlabs/pluto";
import { deep, errors, type record, strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";
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

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "Workspace" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({
      state: { nodes, setNodes, getResource },
      selection: { resourceIDs },
    }) => {
      const resources = getResource(resourceIDs);
      if (!(await confirm(resources))) throw new errors.Canceled();
      const prevNodes = Tree.deepCopy(nodes);
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => ontology.idToString(id)),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({
      selection: { resourceIDs },
      client,
      store,
      state: { getResource },
    }) => {
      const resources = getResource(resourceIDs);
      await client.workspaces.delete(resources.map(({ id }) => id.key));
      const s = store.getState();
      const activeKey = selectActiveKey(s);
      const active = resources.find((r) => r.id.key === activeKey);
      if (active != null) {
        store.dispatch(setActive(null));
        store.dispatch(Layout.clearWorkspace());
      }
    },
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(e)) return;
      handleError(e, "Failed to delete workspace");
    },
  }).mutate;
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

const useCreateSchematic = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const maybeChangeWorkspace = useMaybeChangeWorkspace();
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      selection,
      placeLayout,
      state: { nodes, setNodes, setResource },
      client,
    }) => {
      const workspaceID = selection.resourceIDs[0];
      const schematic = await client.workspaces.schematic.create(workspaceID.key, {
        name: "New Schematic",
        snapshot: false,
        data: deep.copy(Schematic.ZERO_STATE) as unknown as record.Unknown,
      });
      const otg = await client.ontology.retrieve(
        clientSchematic.ontologyID(schematic.key),
      );
      await maybeChangeWorkspace(workspaceID.key);
      placeLayout(
        Schematic.create({
          ...schematic.data,
          key: schematic.key,
          name: schematic.name,
          snapshot: schematic.snapshot,
        }),
      );
      setResource(otg);
      const nextNodes = Tree.setNode({
        tree: nodes,
        destination: ontology.idToString(workspaceID),
        additions: otg,
      });
      setNodes([...nextNodes]);
    },
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      handleError(e, "Failed to create schematic");
    },
  }).mutate;
};

const useCreateLinePlot = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const maybeChangeWorkspace = useMaybeChangeWorkspace();
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      selection,
      placeLayout,
      state: { nodes, setNodes, setResource },
      client,
    }) => {
      const workspaceID = selection.resourceIDs[0];
      const linePlot = await client.workspaces.linePlot.create(workspaceID.key, {
        name: "New Line Plot",
        data: deep.copy(LinePlot.ZERO_SLICE_STATE),
      });
      const otg = await client.ontology.retrieve(
        clientLinePlot.ontologyID(linePlot.key),
      );
      await maybeChangeWorkspace(workspaceID.key);
      placeLayout(LinePlot.create({ ...linePlot.data, ...linePlot }));
      setResource(otg);
      const nextNodes = Tree.setNode({
        tree: nodes,
        destination: ontology.idToString(workspaceID),
        additions: { key: ontology.idToString(otg.id) },
      });
      setNodes([...nextNodes]);
    },
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      handleError(e, "Failed to create line plot");
    },
  }).mutate;
};

const useCreateLog = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const maybeChangeWorkspace = useMaybeChangeWorkspace();
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      selection,
      placeLayout,
      state: { nodes, setNodes, setResource },
      client,
    }) => {
      const workspaceID = selection.resourceIDs[0];
      const log = await client.workspaces.log.create(workspaceID.key, {
        name: "New Log",
        data: deep.copy(Log.ZERO_STATE),
      });
      const otg = await client.ontology.retrieve(clientLog.ontologyID(log.key));
      await maybeChangeWorkspace(workspaceID.key);
      placeLayout(Log.create({ ...log.data, key: log.key, name: log.name }));
      setResource(otg);
      const nextNodes = Tree.setNode({
        tree: nodes,
        destination: ontology.idToString(workspaceID),
        additions: { key: ontology.idToString(otg.id) },
      });
      setNodes([...nextNodes]);
    },
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      handleError(e, "Failed to create log");
    },
  }).mutate;
};

const useCreateTable = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const maybeChangeWorkspace = useMaybeChangeWorkspace();
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      selection,
      placeLayout,
      state: { nodes, setNodes, setResource },
      client,
    }) => {
      const workspaceID = selection.resourceIDs[0];
      const table = await client.workspaces.table.create(workspaceID.key, {
        name: "New Table",
        data: deep.copy(Table.ZERO_STATE),
      });
      const otg = await client.ontology.retrieve(clientTable.ontologyID(table.key));
      await maybeChangeWorkspace(workspaceID.key);
      placeLayout(Table.create({ ...table.data, key: table.key, name: table.name }));
      setResource(otg);
      const nextNodes = Tree.setNode({
        tree: nodes,
        destination: ontology.idToString(workspaceID),
        additions: { key: ontology.idToString(otg.id) },
      });
      setNodes([...nextNodes]);
    },
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      handleError(e, "Failed to create table");
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    selection,
    selection: { resourceIDs, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete();
  const group = Group.useCreateFromSelection();
  const createPlot = useCreateLinePlot();
  const createLog = useCreateLog();
  const createTable = useCreateTable();
  const firstID = selection.resourceIDs[0];
  const importPlot = LinePlotServices.useImport(firstID.key);
  const createSchematic = useCreateSchematic();
  const importSchematic = SchematicServices.useImport(firstID.key);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = useExport(EXTRACTORS);
  const importLog = LogServices.useImport(firstID.key);
  const importTable = TableServices.useImport(firstID.key);
  const resources = getResource(resourceIDs);
  const first = resources[0];
  const handleSelect = {
    delete: () => handleDelete(props),
    rename: () => Text.edit(ontology.idToString(first.id)),
    group: () => group(props),
    createLog: () => createLog(props),
    createPlot: () => createPlot(props),
    createTable: () => createTable(props),
    importPlot,
    importLog,
    importTable,
    createSchematic: () => createSchematic(props),
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
      <Group.MenuItem resourceIDs={resourceIDs} shape={shape} rootID={rootID} />
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
