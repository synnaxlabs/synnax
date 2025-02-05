// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { linePlot, ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Export } from "@/export";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
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
    onError: (err, { state: { setNodes }, handleException }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(err)) return;
      handleException(err, "Failed to delete line plot");
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { resources },
  } = props;
  const del = useDelete();
  const handleLink = Link.useCopyToClipboard();
  const handleExport = LinePlot.useExport();
  const group = Group.useCreateFromSelection();
  const onSelect = {
    delete: () => del(props),
    rename: () => Tree.startRenaming(resources[0].key),
    link: () =>
      handleLink({ name: resources[0].name, ontologyID: resources[0].id.payload }),
    export: () => handleExport(resources[0].id.key),
    group: () => group(props),
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
      <Group.GroupMenuItem selection={selection} />
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      <PMenu.Divider />
      {isSingle && (
        <>
          <Export.MenuItem />
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
    LinePlot.create({ ...linePlot.data, key: linePlot.key, name: linePlot.name }),
  );
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
  handleException,
}): void => {
  client.workspaces.linePlot
    .retrieve(id.key)
    .then((linePlot) => {
      placeLayout(
        LinePlot.create({
          ...linePlot.data,
          key: linePlot.key,
          name: linePlot.name,
          location: "mosaic",
          tab: { mosaicKey: nodeKey, location },
        }),
      );
    })
    .catch((e) => handleException(e, "Failed to load line plot"));
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.BASE_SERVICE,
  type: linePlot.ONTOLOGY_TYPE,
  icon: <Icon.Visualize />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [{ type: Mosaic.HAUL_CREATE_TYPE, key: id.toString() }],
  allowRename: () => true,
  onRename: handleRename,
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
