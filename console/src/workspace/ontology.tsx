// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { deep, errors, type UnknownRecord } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { LinePlotServices } from "@/lineplot/services";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { Schematic } from "@/schematic";
import { SchematicServices } from "@/schematic/services";
import { selectActiveKey } from "@/workspace/selectors";
import { add, rename, setActive } from "@/workspace/slice";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "Workspace" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ state: { nodes, setNodes }, selection: { resources } }) => {
      if (!(await confirm(resources))) throw errors.CANCELED;
      const prevNodes = Tree.deepCopy(nodes);
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => id.toString()),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ selection: { resources }, client, store }) => {
      await client.workspaces.delete(...resources.map(({ id }) => id.key));
      const s = store.getState();
      const activeKey = selectActiveKey(s);
      const active = resources.find((r) => r.id.key === activeKey);
      if (active != null) {
        store.dispatch(setActive(null));
        store.dispatch(Layout.clearWorkspace());
      }
    },
    onError: (e, { addStatus, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(e)) return;
      addStatus({
        variant: "error",
        message: "Failed to delete workspace.",
        description: e.message,
      });
    },
  }).mutate;
};

const useCreateSchematic = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      selection,
      placeLayout,
      services,
      state: { resources, setResources, nodes, setNodes },
      client,
    }) => {
      const workspace = selection.resources[0].id.key;
      const schematic = await client.workspaces.schematic.create(workspace, {
        name: "New Schematic",
        snapshot: false,
        data: deep.copy(Schematic.ZERO_STATE) as unknown as UnknownRecord,
      });
      const otg = await client.ontology.retrieve(
        new ontology.ID({ key: schematic.key, type: "schematic" }),
      );
      placeLayout(
        Schematic.create({
          ...(schematic.data as unknown as Schematic.State),
          key: schematic.key,
          name: schematic.name,
          snapshot: schematic.snapshot,
        }),
      );
      setResources([...resources, otg]);
      const nextNodes = Tree.setNode({
        tree: nodes,
        destination: selection.resources[0].key,
        additions: Ontology.toTreeNodes(services, [otg]),
      });
      setNodes([...nextNodes]);
    },
    onError: (e, { addStatus, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      addStatus({
        variant: "error",
        message: "Failed to create schematic.",
        description: e.message,
      });
    },
  }).mutate;

const useCreateLinePlot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      selection,
      placeLayout,
      services,
      state: { setResources, resources, nodes, setNodes },
      client,
    }) => {
      const workspace = selection.resources[0].id.key;
      const linePlot = await client.workspaces.linePlot.create(workspace, {
        name: "New Line Plot",
        data: deep.copy(LinePlot.ZERO_SLICE_STATE) as unknown as UnknownRecord,
      });
      const otg = await client.ontology.retrieve(
        new ontology.ID({ key: linePlot.key, type: "lineplot" }),
      );
      placeLayout(
        LinePlot.create({
          ...(linePlot.data as unknown as LinePlot.SliceState),
          key: linePlot.key,
          name: linePlot.name,
        }),
      );
      setResources([...resources, otg]);
      const nextNodes = Tree.setNode({
        tree: nodes,
        destination: selection.resources[0].key,
        additions: Ontology.toTreeNodes(services, [otg]),
      });
      setNodes([...nextNodes]);
    },
    onError: (e, { addStatus, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      addStatus({
        variant: "error",
        message: "Failed to create line plot.",
        description: e.message,
      });
    },
  }).mutate;

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    selection,
    selection: { resources },
  } = props;
  const del = useDelete();
  const createSchematic = useCreateSchematic();
  const createLinePlot = useCreateLinePlot();
  const importLinePlot = LinePlot.useImport();
  const group = Group.useCreateFromSelection();
  const handleLink = Link.useCopyToClipboard();
  const importSchematic = Schematic.useImport(selection.resources[0].id.key);
  const handleSelect = {
    delete: () => del(props),
    rename: () => Tree.startRenaming(resources[0].id.toString()),
    group: () => group(props),
    plot: () => createLinePlot(props),
    importLinePlot: () => importLinePlot(),
    schematic: () => createSchematic(props),
    importSchematic: () => importSchematic(),
    link: () =>
      handleLink({
        name: resources[0].name,
        ontologyID: resources[0].id.payload,
      }),
  };
  const singleResource = resources.length === 1;
  const canCreateSchematic = Schematic.useSelectHasPermission();
  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      {singleResource && (
        <>
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.DeleteItem />
      <Group.GroupMenuItem selection={selection} />
      <PMenu.Divider />
      {singleResource && (
        <>
          <PMenu.Item itemKey="plot" startIcon={<LinePlotServices.CreateIcon />}>
            Create New Line Plot
          </PMenu.Item>
          <PMenu.Item
            itemKey="importLinePlot"
            startIcon={<LinePlotServices.ImportIcon />}
          >
            Import Line Plot
          </PMenu.Item>
          {canCreateSchematic && (
            <>
              <PMenu.Item
                itemKey="schematic"
                startIcon={<SchematicServices.CreateIcon />}
              >
                Create New Schematic
              </PMenu.Item>
              <PMenu.Item
                itemKey="importSchematic"
                startIcon={<SchematicServices.ImportIcon />}
              >
                Import Schematic
              </PMenu.Item>
            </>
          )}
          <PMenu.Divider />
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleSelect: Ontology.HandleSelect = ({ selection, client, store }) => {
  void (async () => {
    const workspace = await client.workspaces.retrieve(selection[0].id.key);
    store.dispatch(add({ workspaces: [workspace] }));
    store.dispatch(
      Layout.setWorkspace({
        slice: workspace.layout as unknown as Layout.SliceState,
        keepNav: false,
      }),
    );
  })();
};

const handleRename: Ontology.HandleTreeRename = {
  eager: ({ id, name, store }) => store.dispatch(rename({ key: id.key, name })),
  execute: async ({ client, id, name }) => await client.workspaces.rename(id.key, name),
  rollback: async ({ id, store }, prevName) =>
    store.dispatch(rename({ key: id.key, name: prevName })),
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "workspace",
  icon: <Icon.Workspace />,
  hasChildren: true,
  canDrop: () => false,
  TreeContextMenu,
  onSelect: handleSelect,
  haulItems: () => [],
  allowRename: () => true,
  onRename: handleRename,
};
