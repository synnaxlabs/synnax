// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Mosaic, Text, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Export } from "@/export";
import { Group } from "@/group";
import { Common } from "@/hardware/common";
import { type LayoutArgs } from "@/hardware/common/task/Task";
import { createLayout, retrieveAndPlaceLayout } from "@/hardware/task/layouts";
import { useRangeSnapshot } from "@/hardware/task/useRangeSnapshot";
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
    onMutate: async ({
      state: { nodes, setNodes, getResource },
      selection: { resourceIDs },
    }) => {
      const prevNodes = Tree.deepCopy(nodes);
      const resources = resourceIDs.map((id) => getResource(id));
      if (!(await confirm(resources))) throw new errors.Canceled();
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => ontology.idToString(id)),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async (props: Ontology.TreeContextMenuProps) => {
      const {
        client,
        selection: { resourceIDs },
        removeLayout,
      } = props;
      const keys = resourceIDs.map((id) => id.key);
      await client.hardware.tasks.delete(keys);
      removeLayout(...keys);
    },
    onError: (
      e: Error,
      { handleError, selection: { resourceIDs }, state: { getResource } },
    ) => {
      let message = "Failed to delete tasks";
      if (resourceIDs.length === 1)
        message = `Failed to delete task ${getResource(resourceIDs[0]).name}`;
      if (errors.Canceled.matches(e)) return;
      handleError(e, message);
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    store,
    selection,
    client,
    addStatus,
    handleError,
    state: { getResource, shape },
  } = props;
  const { resourceIDs } = selection;
  const resources = getResource(resourceIDs);
  const del = useDelete();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = Common.Task.useExport();
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
    rename: () => Text.edit(resourceIDs[0].key),
    link: () => handleLink({ name: resources[0].name, ontologyID: resources[0].id }),
    export: () => handleExport(resourceIDs[0].key),
    rangeSnapshot: () => snap(resources),
    group: () => group(props),
  };
  const singleResource = resourceIDs.length === 1;
  const hasNoSnapshots = resources.every((r) => r.data?.snapshot === false);
  return (
    <PMenu.Menu level="small" gap="small" onChange={onSelect}>
      <Group.MenuItem resourceIDs={resourceIDs} shape={shape} />
      {hasNoSnapshots && range?.persisted === true && (
        <>
          <Range.SnapshotMenuItem key="snapshot" range={range} />
          <PMenu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <PMenu.Item itemKey="edit">
            <Icon.Edit />
            {`${resources[0].data?.snapshot ? "View" : "Edit"} Configuration`}
          </PMenu.Item>
          <Menu.RenameItem />
          <Link.CopyMenuItem />
          <Export.MenuItem />
          <PMenu.Divider />
        </>
      )}
      <PMenu.Item itemKey="delete">
        <Icon.Delete />
        Delete
      </PMenu.Item>
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name, store }) => {
    const task = await client.hardware.tasks.retrieve({ key: id.key });
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
}) =>
  handleError(async () => {
    const task = await client.hardware.tasks.retrieve({ key: id.key });
    const layout = createLayout(task);
    placeLayout({ ...layout, tab: { mosaicKey: nodeKey, location } });
  }, "Failed to load task layout");

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "task",
  icon: <Icon.Task />,
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
