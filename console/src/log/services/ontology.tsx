// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, ontology, type Synnax } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Mosaic, Text, Tree } from "@synnaxlabs/pluto";
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
    onMutate: async ({
      selection: { resourceIDs },
      removeLayout,
      state: { nodes, setNodes, getResource },
    }) => {
      const resources = getResource(resourceIDs);
      if (!(await confirm(resources))) throw new errors.Canceled();
      const ids = ontology.parseIDs(resourceIDs);
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
      await client.workspaces.log.delete(ids.map((id) => id.key));
    },
    onError: (err, { state: { setNodes }, handleError }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(err)) return;
      handleError(err, "Failed to delete log");
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
  const handleExport = Log.useExport();
  const group = Group.useCreateFromSelection();
  const firstID = resourceIDs[0];
  const firstResource = getResource(firstID);
  const onSelect = useAsyncActionMenu({
    delete: () => del(props),
    rename: () => Text.edit(ontology.idToString(firstID)),
    link: () =>
      handleLink({
        name: firstResource.name,
        ontologyID: resourceIDs[0],
      }),
    export: () => handleExport(resourceIDs[0].key),
    group: () => group(props),
  });
  const isSingle = resourceIDs.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" gap="small">
      <Menu.RenameItem />
      <Menu.DeleteItem />
      <Group.MenuItem resourceIDs={resourceIDs} shape={shape} />
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
  handleError,
}) => {
  loadLog(client, selection[0].id, placeLayout).catch((e) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "log",
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
}) =>
  handleError(async () => {
    const log = await client.workspaces.log.retrieve(id.key);
    placeLayout(
      Log.create({
        name: log.name,
        ...log.data,
        key: id.key,
        location: "mosaic",
        tab: { mosaicKey: nodeKey, location },
      }),
    );
  }, "Failed to load log");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: log.ONTOLOGY_TYPE,
  icon: <Icon.Log />,
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
