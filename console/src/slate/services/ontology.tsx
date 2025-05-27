// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, slate, type Synnax } from "@synnaxlabs/client";
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
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { Range } from "@/range";
import { Slate } from "@/slate";
import { translateSlateForward } from "@/slate/types/translate";

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "slate" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ selection, removeLayout, state: { nodes, setNodes } }) => {
      if (!(await confirm(selection.resources))) throw new errors.Canceled();
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
      await client.slates.delete(ids.map((id) => id.key));
    },
    onError: (err, { state: { setNodes }, handleError }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(err)) return;
      handleError(err, "Failed to delete slate");
    },
  }).mutate;
};

// const useCopy = (): ((props: Ontology.TreeContextMenuProps) => void) =>
//   useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
//     mutationFn: async ({
//       client,
//       selection: { resources, parentID },
//       state,
//       services,
//     }) => {
//       if (parentID == null) return;
//       const slates = await Promise.all(
//         resources.map(
//           async (res) =>
//             await client.slates.copy(res.id.key, `${res.name} (copy)`, false),
//         ),
//       );
//       const otgIDs = slates.map(({ key }) => slate.ontologyID(key));
//       const otg = await client.ontology.retrieve(otgIDs);
//       state.setResources([...state.resources, ...otg]);
//       const nextTree = Tree.setNode({
//         tree: state.nodes,
//         destination: parentID.toString(),
//         additions: Ontology.toTreeNodes(services, otg),
//       });
//       state.setNodes([...nextTree]);
//       Tree.startRenaming(otg[0].id.toString());
//     },
//     onError: (err, { handleError }) => {
//       handleError(err, "Failed to copy slate");
//     },
//   }).mutate;

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { resources },
  } = props;
  const activeRange = Range.useSelect();
  const del = useDelete();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const group = Group.useCreateFromSelection();
  const onSelect = useAsyncActionMenu({
    delete: () => del(props),
    rename: () => Tree.startRenaming(resources[0].key),
    group: () => group(props),
    link: () =>
      handleLink({ name: resources[0].name, ontologyID: resources[0].id.payload }),
  });
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Menu.RenameItem />
      <Menu.DeleteItem />
      <Group.MenuItem selection={selection} />
      <PMenu.Divider />
      {resources.every((r) => r.data?.snapshot === false) && (
        <>
          <Range.SnapshotMenuItem range={activeRange} />
          <PMenu.Item itemKey="copy" startIcon={<Icon.Copy />}>
            Copy
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
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
  execute: async ({ client, id, name }) => {},
  rollback: ({ id: { key }, name, store }) =>
    store.dispatch(Layout.rename({ key, name })),
};

const loadSlate = async (
  client: Synnax,
  id: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const slate = await client.slates.retrieve(id.key);
  placeLayout(Slate.create(translateSlateForward(slate)));
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  loadSlate(client, selection[0].id, placeLayout).catch((e) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "slate",
    );
    handleError(e, `Failed to select ${names}`);
  });
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  placeLayout,
  handleError,
}) => {
  client.slates
    .retrieve(id.key)
    .then((slate) => placeLayout(Slate.create(translateSlateForward(slate))))
    .catch((e) => handleError(e, "Failed to load slate"));
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: slate.ONTOLOGY_TYPE,
  icon: <Icon.Slate />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [{ type: Mosaic.HAUL_CREATE_TYPE, key: id.toString() }],
  allowRename: () => true,
  onRename: handleRename,
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
