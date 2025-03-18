// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, ranger, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Group } from "@/group";
import { type LayoutArgs } from "@/hardware/common/task/Task";
import { createLayout, retrieveAndPlaceLayout } from "@/hardware/task/layouts";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { Range } from "@/range";

const handleSelect: Ontology.HandleSelect = ({
  selection,
  placeLayout,
  client,
  handleError,
}) => {
  if (selection.length === 0) return;
  const key = selection[0].id.key;
  const name = selection[0].name;
  handleError(
    async () => await retrieveAndPlaceLayout(client, key, placeLayout),
    `Could not open ${name}`,
  );
};

const useDelete = () => {
  const confirm = Ontology.useConfirmDelete({ type: "Task" });
  return useMutation({
    onMutate: async ({ state: { nodes, setNodes }, selection: { resources } }) => {
      const prevNodes = Tree.deepCopy(nodes);
      if (!(await confirm(resources))) throw errors.CANCELED;
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => id.toString()),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async (props: Ontology.TreeContextMenuProps) => {
      const {
        client,
        selection: { resources },
        removeLayout,
      } = props;
      await client.hardware.tasks.delete(resources.map(({ id }) => BigInt(id.key)));
      removeLayout(...resources.map(({ id }) => id.key));
    },
    onError: (e: Error, { handleError, selection: { resources } }) => {
      let message = "Failed to delete tasks";
      if (resources.length === 1)
        message = `Failed to delete task ${resources[0].name}`;
      if (errors.CANCELED.matches(e)) return;
      handleError(e, message);
    },
  }).mutate;
};

const useRangeSnapshot = () =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ store, client, selection: { resources, parent } }) => {
      const activeRange = Range.selectActiveKey(store.getState());
      if (activeRange === null || parent == null) return;
      const tasks = await Promise.all(
        resources.map(({ id, name }) =>
          client.hardware.tasks.copy(id.key, `${name} (Snapshot)`, true),
        ),
      );
      const otgIDs = tasks.map((t) => t.ontologyID);
      const rangeID = ranger.ontologyID(activeRange);
      await client.ontology.moveChildren(
        new ontology.ID(parent.key),
        rangeID,
        ...otgIDs,
      );
    },
    onError: (e: Error, { handleError }) => handleError(e, "Failed to create snapshot"),
  }).mutate;

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { store, selection, client, addStatus, handleError } = props;
  const { resources, nodes } = selection;
  const del = useDelete();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const snap = useRangeSnapshot();
  const range = Range.useSelect();
  const group = Group.useCreateFromSelection();
  const onSelect = {
    delete: () => del(props),
    edit: () =>
      handleSelect({
        selection: resources,
        placeLayout: props.placeLayout,
        client,
        addStatus,
        store,
        handleError,
        removeLayout: props.removeLayout,
        services: props.services,
      }),
    rename: () => Tree.startRenaming(nodes[0].key),
    link: () =>
      handleLink({ name: resources[0].name, ontologyID: resources[0].id.payload }),
    rangeSnapshot: () => snap(props),
    group: () => group(props),
  };
  const singleResource = resources.length === 1;
  const hasNoSnapshots = resources.every((r) => r.data?.snapshot === false);
  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={onSelect}>
      <Group.MenuItem selection={selection} />
      {hasNoSnapshots && range?.persisted === true && (
        <>
          <Range.SnapshotMenuItem key="snapshot" range={range} />
          <PMenu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <PMenu.Item itemKey="edit" startIcon={<Icon.Edit />}>
            {`${resources[0].data?.snapshot ? "View" : "Edit"} Configuration`}
          </PMenu.Item>
          <Menu.RenameItem />
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name, store }) => {
    const task = await client.hardware.tasks.retrieve(id.key);
    await client.hardware.tasks.create({ ...task, name });
    const layout = Layout.selectByFilter(
      store.getState(),
      (l) => (l.args as LayoutArgs)?.taskKey === id.key,
    );
    if (layout == null) return;
    store.dispatch(Layout.rename({ key: layout.key, name }));
  },
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = ({
  client,
  id,
  placeLayout,
  nodeKey,
  location,
  handleError,
}) => {
  client.hardware.tasks
    .retrieve(id.key)
    .then((task) => {
      const layout = createLayout(task);
      placeLayout({ ...layout, tab: { mosaicKey: nodeKey, location } });
    })
    .catch(handleError);
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: task.ONTOLOGY_TYPE,
  icon: <Icon.Task />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [{ type: Mosaic.HAUL_CREATE_TYPE, key: id.toString() }],
  allowRename: () => true,
  onRename: handleRename,
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
};
