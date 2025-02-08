// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Ontology } from "@/ontology";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = Ontology.useConfirmDelete({ type: "Rack" });
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
    mutationFn: async ({ selection: { resources }, client }) =>
      await client.hardware.racks.delete(resources.map(({ id }) => Number(id.key))),
    onError: (e, { handleException, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(e)) return;
      handleException(e, "Failed to delete racks");
    },
  }).mutate;
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) => {
    const rack = await client.hardware.racks.retrieve(id.key);
    await client.hardware.racks.create({ ...rack, name });
  },
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { selection } = props;
  const { nodes } = selection;
  const handleDelete = useDelete();
  const onSelect = {
    rename: () => Tree.startRenaming(nodes[0].key),
    delete: () => handleDelete(props),
  };
  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={onSelect}>
      <Menu.RenameItem />
      <Menu.DeleteItem />
    </PMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: rack.ONTOLOGY_TYPE,
  icon: <Icon.Rack />,
  hasChildren: true,
  canDrop: () => false,
  onSelect: () => {},
  haulItems: () => [],
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
};
