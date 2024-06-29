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
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";

import { Cluster } from "@/cluster";
import { Menu } from "@/components/menu";
import { Layout } from "@/layout";
import { create } from "@/lineplot/LinePlot";
import { type State } from "@/lineplot/slice";
import { Link } from "@/link";
import { Ontology } from "@/ontology";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ selection, removeLayout, state: { nodes, setNodes } }) => {
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
      addStatus({
        variant: "error",
        message: "Failed to delete line plot",
        description: err.message,
      });
    },
  }).mutate;

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { resources } = props.selection;
  const del = useDelete();
  const activeKey = Cluster.useSelectActiveKey();
  const onSelect = {
    delete: () => del(props),
    rename: () => Tree.startRenaming(resources[0].key),
    link: () => {
      const toCopy = `synnax://cluster/${activeKey}/lineplot/${resources[0].id.key}`;
      navigator.clipboard.writeText(toCopy);
    },
  };
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      {isSingle && (
        <>
          <Ontology.RenameMenuItem />
          <PMenu.Divider />
        </>
      )}
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      <PMenu.Divider />
      {isSingle && <Link.CopyMenuItem />}
      <PMenu.Divider />
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

const handleMosaicDrop: Ontology.HandleMosaicDrop = async ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
}): Promise<void> => {
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
