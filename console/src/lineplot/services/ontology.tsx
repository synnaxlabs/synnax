// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { linePlot, ontology } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Mosaic, Text, Tree } from "@synnaxlabs/pluto";
import { errors, strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
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
    onMutate: async ({
      selection,
      removeLayout,
      state: { nodes, setNodes, getResource },
    }) => {
      if (!(await confirm(getResource(selection.resourceIDs))))
        throw new errors.Canceled();
      const ids = ontology.parseIDs(selection.resourceIDs);
      const keys = ids.map((id) => id.key);
      removeLayout(...keys);
      const prevNodes = Tree.deepCopy(nodes);
      const next = Tree.removeNode({
        tree: nodes,
        keys: ids.map((id) => ontology.idToString(id)),
      });
      setNodes([...next]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection }) => {
      const ids = ontology.parseIDs(selection.resourceIDs);
      await new Promise((resolve) => setTimeout(resolve, 1000));
      await client.workspaces.linePlot.delete(ids.map((id) => id.key));
    },
    onError: (err, { state: { setNodes }, handleError }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(err)) return;
      handleError(err, "Failed to delete line plot");
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resourceIDs },
    state: { getResource, shape },
  } = props;
  const del = useDelete();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = LinePlot.useExport();
  const group = Group.useCreateFromSelection();
  const firstID = resourceIDs[0];
  const isSingle = resourceIDs.length === 1;
  const first = getResource(firstID);
  const onSelect = {
    delete: () => del(props),
    rename: () => Text.edit(ontology.idToString(firstID)),
    link: () => handleLink({ name: first.name, ontologyID: firstID }),
    export: () => handleExport(first.id.key),
    group: () => group(props),
  };
  return (
    <PMenu.Menu onChange={onSelect} level="small" gap="small">
      {isSingle && (
        <>
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      <Group.MenuItem resourceIDs={resourceIDs} shape={shape} />
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

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  client.workspaces.linePlot
    .retrieve(selection[0].id.key)
    .then((linePlot) => {
      placeLayout(
        LinePlot.create({ ...linePlot.data, key: linePlot.key, name: linePlot.name }),
      );
    })
    .catch((e) => {
      const names = strings.naturalLanguageJoin(
        selection.map(({ name }) => name),
        "line plot",
      );
      handleError(e, `Failed to select ${names}`);
    });
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
  handleError,
}): void =>
  handleError(async () => {
    const linePlot = await client.workspaces.linePlot.retrieve(id.key);
    placeLayout(
      LinePlot.create({
        ...linePlot.data,
        key: linePlot.key,
        name: linePlot.name,
        location: "mosaic",
        tab: { mosaicKey: nodeKey, location },
      }),
    );
  }, "Failed to load line plot");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: linePlot.ONTOLOGY_TYPE,
  icon: <Icon.LinePlot />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [
    { type: Mosaic.HAUL_CREATE_TYPE, key: ontology.idToString(id) },
  ],
  allowRename: () => true,
  onRename: handleRename,
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
