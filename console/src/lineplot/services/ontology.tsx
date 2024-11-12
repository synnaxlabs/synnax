// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Layout } from "@/layout";
import { useExport } from "@/lineplot/file";
import { create } from "@/lineplot/LinePlot";
import { type State } from "@/lineplot/slice";
import { Link } from "@/link";
import { type Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "LinePlot" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ selection, removeLayout, state: { nodes, setNodes } }) => {
      if (!(await confirm(selection.resources))) throw errors.CANCELED;
      const ids = selection.resources.map((res) => new ontology.ID(res.key));
      const keys = ids.map((id) => id.key);
      removeLayout(...keys);
      const prevNodes = Tree.deepCopy(nodes);
      const next = Tree.removeNode({
        tree: nodes,
        keys: ids.map((id) => id.toString()),
      });
      setNodes([...next]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection }) => {
      const ids = selection.resources.map((res) => new ontology.ID(res.key));
      await new Promise((resolve) => setTimeout(resolve, 1000));
      await client.workspaces.linePlot.delete(ids.map((id) => id.key));
    },
    onError: (err, { state: { setNodes }, addStatus }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(err)) return;
      addStatus({
        variant: "error",
        message: "Failed to delete line plot",
        description: err.message,
      });
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { resources } = props.selection;
  const del = useDelete();
  const handleLink = Link.useCopyToClipboard();
  const handleExport = useExport(resources[0].name);
  const onSelect = {
    delete: () => del(props),
    rename: () => Tree.startRenaming(resources[0].key),
    link: () =>
      handleLink({
        name: resources[0].name,
        ontologyID: resources[0].id.payload,
      }),
    export: () => handleExport(resources[0].id.key),
  };
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      {isSingle && (
        <>
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      <PMenu.Divider />
      {isSingle && (
        <>
          <PMenu.Item itemKey="export" startIcon={<Icon.Export />}>
            Export
          </PMenu.Item>
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = {
  eager: ({ store, id, name }) => store.dispatch(Layout.rename({ key: id.key, name })),
  execute: async ({ client, id, name }) =>
    await client.workspaces.linePlot.rename(id.key, name),
  rollback: ({ store, id }, prevName) =>
    store.dispatch(Layout.rename({ key: id.key, name: prevName })),
};

const handleSelect: Ontology.HandleSelect = async ({
  client,
  selection,
  placeLayout,
}): Promise<void> => {
  const linePlot = await client.workspaces.linePlot.retrieve(selection[0].id.key);
  placeLayout(
    create({
      ...(linePlot.data as unknown as State),
      key: linePlot.key,
      name: linePlot.name,
    }),
  );
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
  addStatus,
}): void => {
  void (async () => {
    try {
      const linePlot = await client.workspaces.linePlot.retrieve(id.key);
      placeLayout(
        create({
          ...(linePlot.data as unknown as State),
          key: linePlot.key,
          name: linePlot.name,
          location: "mosaic",
          tab: {
            mosaicKey: nodeKey,
            location,
          },
        }),
      );
    } catch (err) {
      addStatus({
        variant: "error",
        message: "Failed to load line plot",
        description: (err as Error).message,
      });
    }
  })();
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "lineplot",
  icon: <Icon.Visualize />,
  hasChildren: false,
  haulItems: (r) => [
    {
      type: Mosaic.HAUL_CREATE_TYPE,
      key: r.id.toString(),
    },
  ],
  allowRename: () => true,
  onRename: handleRename,
  canDrop: () => false,
  TreeContextMenu,
  onMosaicDrop: handleMosaicDrop,
  onSelect: handleSelect,
};
