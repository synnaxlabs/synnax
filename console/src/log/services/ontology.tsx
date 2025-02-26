// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, ontology, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { errors, strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { useAsyncActionMenu } from "@/hooks/useAsyncAction";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Log } from "@/log";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "Log" });
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
      await client.workspaces.log.delete(ids.map((id) => id.key));
    },
    onError: (err, { state: { setNodes }, handleException }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(err)) return;
      handleException(err, "Failed to delete log");
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { resources },
  } = props;
  const del = useDelete();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = Log.useExport();
  const group = Group.useCreateFromSelection();
  const onSelect = useAsyncActionMenu({
    delete: () => del(props),
    rename: () => Tree.startRenaming(resources[0].key),
    link: () =>
      handleLink({ name: resources[0].name, ontologyID: resources[0].id.payload }),
    export: () => handleExport(resources[0].id.key),
    group: () => group(props),
  });
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Menu.RenameItem />
      <Menu.DeleteItem />
      <Group.MenuItem selection={selection} />
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
  eager: ({ id: { key }, name, store }) => store.dispatch(Layout.rename({ key, name })),
  execute: async ({ client, id, name }) =>
    await client.workspaces.log.rename(id.key, name),
  rollback: ({ id: { key }, name, store }) =>
    store.dispatch(Layout.rename({ key, name })),
};

const loadLog = async (client: Synnax, id: ontology.ID, placeLayout: Layout.Placer) => {
  const log = await client.workspaces.log.retrieve(id.key);
  placeLayout(Log.create({ ...(log.data as Log.State), key: log.key, name: log.name }));
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleException,
}) => {
  loadLog(client, selection[0].id, placeLayout).catch((e) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "log",
    );
    handleException(e, `Failed to select ${names}`);
  });
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
  handleException,
}) => {
  client.workspaces.log
    .retrieve(id.key)
    .then((log) => {
      placeLayout(
        Log.create({
          name: log.name,
          ...log.data,
          key: id.key,
          location: "mosaic",
          tab: { mosaicKey: nodeKey, location },
        }),
      );
    })
    .catch((e) => handleException(e, "Failed to load log"));
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: log.ONTOLOGY_TYPE,
  icon: <Icon.Log />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [{ type: Mosaic.HAUL_CREATE_TYPE, key: id.toString() }],
  allowRename: () => true,
  onRename: handleRename,
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
