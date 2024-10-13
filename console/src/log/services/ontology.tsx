// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { useAsyncActionMenu } from "@/hooks/useAsyncAction";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Log } from "@/log";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { Range } from "@/range";

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
    onError: (err, { state: { setNodes }, addStatus }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.CANCELED.matches(err)) return;
      addStatus({
        variant: "error",
        message: "Failed to delete log",
        description: err.message,
      });
    },
  }).mutate;
};

const useCopy = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({
      client,
      selection: { resources, parent },
      state,
      services,
    }) => {
      if (parent == null) return;
      const logs = await Promise.all(
        resources.map(
          async (res) =>
            await client.workspaces.log.copy(res.id.key, res.name + " (copy)", false),
        ),
      );
      const otgIDs = logs.map(({ key }) => new ontology.ID({ type: "log", key }));
      const otg = await client.ontology.retrieve(otgIDs);
      state.setResources([...state.resources, ...otg]);
      const nextTree = Tree.setNode({
        tree: state.nodes,
        destination: parent.key,
        additions: Ontology.toTreeNodes(services, otg),
      });
      state.setNodes([...nextTree]);
      Tree.startRenaming(otg[0].id.toString());
    },
    onError: (err, { addStatus }) => {
      addStatus({
        variant: "error",
        message: "Failed to copy log",
        description: err.message,
      });
    },
  }).mutate;

const useRangeSnapshot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    mutationFn: async ({ client, selection: { resources, parent }, store }) => {
      const activeRange = Range.selectActiveKey(store.getState());
      if (activeRange == null || parent == null) return;
      const logs = await Promise.all(
        resources.map(
          async (res) =>
            await client.workspaces.log.copy(res.id.key, res.name + " (snap)", true),
        ),
      );
      const otgIDs = logs.map(({ key }) => new ontology.ID({ type: "log", key }));
      const rangeID = new ontology.ID({ type: "range", key: activeRange });
      await client.ontology.moveChildren(
        new ontology.ID(parent.key),
        rangeID,
        ...otgIDs,
      );
    },
  }).mutate;

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resources },
  } = props;
  const activeRange = Range.useSelect();
  const del = useDelete();
  const copy = useCopy();
  const snapshot = useRangeSnapshot();
  // const handleExport = Log.useExport(resources[0].name);
  const handleLink = Link.useCopyToClipboard();
  const onSelect = useAsyncActionMenu("log.menu", {
    delete: () => del(props),
    copy: () => copy(props),
    rangeSnapshot: () => snapshot(props),
    rename: () => Tree.startRenaming(resources[0].key),
    link: () =>
      handleLink({
        name: resources[0].name,
        ontologyID: resources[0].id.payload,
      }),
  });
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Menu.RenameItem />
      <Menu.DeleteItem />
      <PMenu.Divider />
      {resources.every((r) => r.data?.snapshot === false) && (
        <Range.SnapshotMenuItem range={activeRange} />
      )}
      <PMenu.Item itemKey="copy" startIcon={<Icon.Copy />}>
        Copy
      </PMenu.Item>
      <PMenu.Divider />
      {isSingle && (
        <>
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
  placeLayout(
    Log.create({
      ...(log.data as unknown as Log.State),
      key: log.key,
      name: log.name,
    }),
  );
};

const handleSelect: Ontology.HandleSelect = async ({
  client,
  selection,
  placeLayout,
}) => await loadLog(client, selection[0].id, placeLayout);

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  location,
  nodeKey,
  placeLayout,
}) => {
  void (async () => {
    const log = await client.workspaces.log.retrieve(id.key);
    placeLayout(
      Log.create({
        name: log.name,
        ...(log.data as unknown as Log.State),
        location: "mosaic",
        tab: {
          mosaicKey: nodeKey,
          location,
        },
      }),
    );
  })();
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "log",
  icon: <Icon.Log />,
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
