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
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { Range } from "@/range";

const ZERO_LAYOUT_STATES: Record<string, (create?: boolean) => Layout.State> = {
  [OPC.Task.READ_TYPE]: OPC.Task.configureReadLayout,
  [NI.Task.ANALOG_READ_TYPE]: NI.Task.configureAnalogReadLayout,
  [NI.Task.DIGITAL_WRITE_TYPE]: NI.Task.configureDigitalWriteLayout,
  [NI.Task.DIGITAL_READ_TYPE]: NI.Task.configureDigitalReadLayout,
};

export const createTaskLayout = (key: string, type: string): Layout.State => {
  const baseLayout = ZERO_LAYOUT_STATES[type];
  return {
    ...baseLayout(false),
    key,
  };
};

const handleSelect: Ontology.HandleSelect = ({
  selection,
  placeLayout,
  client,
  addStatus,
}) => {
  if (selection.length === 0) return;
  const task = selection[0].id;
  void (async () => {
    try {
      const t = await client.hardware.tasks.retrieve(task.key);
      console.log(t.type);
      const baseLayout = ZERO_LAYOUT_STATES[t.type];
      placeLayout({
        ...baseLayout(false),
        key: selection[0].id.key,
      });
    } catch (e) {
      addStatus({ variant: "error", message: (e as Error).message });
    }
  })();
};

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "Task" });
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
    onError: (e: Error, { addStatus, selection: { resources } }) => {
      let message = "Failed to delete tasks";
      if (resources.length === 1)
        message = `Failed to delete task ${resources[0].name}`;
      if (errors.CANCELED.matches(e)) return;
      addStatus({
        variant: "error",
        message,
        description: e.message,
      });
    },
  }).mutate;
};

const useRangeSnapshot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ store, client, selection: { resources, parent } }) => {
      const activeRange = Range.selectActiveKey(store.getState());
      if (activeRange === null || parent == null) return;
      const tasks = await Promise.all(
        resources.map(({ id, name }) =>
          client.hardware.tasks.copy(id.key, name + " (Snapshot)", true),
        ),
      );
      const otgIDs = tasks.map((t) => t.ontologyID);
      const rangeID = new ontology.ID({ type: "range", key: activeRange });
      await client.ontology.moveChildren(
        new ontology.ID(parent.key),
        rangeID,
        ...otgIDs,
      );
    },
    onError: (e: Error, { addStatus }) => {
      addStatus({
        variant: "error",
        message: "Failed to create snapshot",
        description: e.message,
      });
    },
  }).mutate;

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { store, selection, client, addStatus } = props;
  const { resources, nodes } = selection;
  const del = useDelete();
  const handleLink = Link.useCopyToClipboard();
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
        removeLayout: props.removeLayout,
        services: props.services,
      }),
    rename: () => Tree.startRenaming(nodes[0].key),
    link: () =>
      handleLink({
        name: resources[0].name,
        ontologyID: resources[0].id.payload,
      }),
    rangeSnapshot: () => snap(props),
    group: () => group(props),
  };
  const singleResource = resources.length === 1;
  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={onSelect}>
      <Group.GroupMenuItem selection={selection} />
      {resources.every((r) => r.data?.snapshot === false) && (
        <Range.SnapshotMenuItem key="snapshot" range={range} />
      )}
      {singleResource && (
        <>
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
  execute: async ({ client, id, name }) => {
    const task = await client.hardware.tasks.retrieve(id.key);
    await client.hardware.tasks.create({ ...task, name });
  },
};

const handleMosaicDrop: Ontology.HandleMosaicDrop = async ({
  client,
  id,
  placeLayout,
  nodeKey,
  location,
}) => {
  const task = await client.hardware.tasks.retrieve(id.key);
  placeLayout({
    ...ZERO_LAYOUT_STATES[task.type](false),
    key: id.key,
    tab: {
      mosaicKey: nodeKey,
      location,
    },
  });
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "task",
  hasChildren: false,
  icon: <Icon.Task />,
  canDrop: () => false,
  onSelect: handleSelect,
  onMosaicDrop: handleMosaicDrop,
  TreeContextMenu,
  haulItems: (r) => [
    {
      type: Mosaic.HAUL_CREATE_TYPE,
      key: r.id.toString(),
    },
  ],
  allowRename: () => true,
  onRename: handleRename,
};
