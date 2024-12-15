// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax, type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Mosaic, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { type LayoutArgs } from "@/hardware/task/common/createLayoutCreator";
import { type Layout } from "@/layout";
import { Link } from "@/link";
import { type Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { Range } from "@/range";

const ZERO_LAYOUT_STATES: Record<
  string,
  (args: LayoutArgs) => Layout.State<LayoutArgs>
> = {
  [LabJack.Task.READ_TYPE]: (args) =>
    LabJack.Task.createReadLayout({
      ...args,
      initialValues: { ...args.initialValues, type: LabJack.Task.READ_TYPE },
    }),
  [LabJack.Task.WRITE_TYPE]: (args) =>
    LabJack.Task.createWriteLayout({
      ...args,
      initialValues: { ...args.initialValues, type: LabJack.Task.WRITE_TYPE },
    }),
  [OPC.Task.READ_TYPE]: (args) =>
    OPC.Task.configureReadLayout({
      ...args,
      initialValues: { ...args.initialValues, type: OPC.Task.READ_TYPE },
    }),
  [OPC.Task.WRITE_TYPE]: (args) =>
    OPC.Task.createWriteLayout({
      ...args,
      initialValues: { ...args.initialValues, type: OPC.Task.WRITE_TYPE },
    }),
  [NI.Task.ANALOG_READ_TYPE]: (args) =>
    NI.Task.createAnalogReadLayout({
      ...args,
      initialValues: { ...args.initialValues, type: NI.Task.ANALOG_READ_TYPE },
    }),
  [NI.Task.DIGITAL_WRITE_TYPE]: (args) =>
    NI.Task.createDigitalWriteLayout({
      ...args,
      initialValues: { ...args.initialValues, type: NI.Task.DIGITAL_WRITE_TYPE },
    }),
  [NI.Task.DIGITAL_READ_TYPE]: (args) =>
    NI.Task.createDigitalReadLayout({
      ...args,
      initialValues: { ...args.initialValues, type: NI.Task.DIGITAL_READ_TYPE },
    }),
};

export const createLayout = (task: task.Task): Layout.State => {
  const configureLayout = ZERO_LAYOUT_STATES[task.type];
  if (configureLayout == null) throw new Error(`No layout configured for ${task.type}`);
  return configureLayout({ create: false, initialValues: task.payload });
};

export const retrieveAndPlaceLayout = async (
  client: Synnax,
  key: task.TaskKey,
  placeLayout: Layout.Placer,
) => {
  const t = await client.hardware.tasks.retrieve(key);
  const layout = createLayout(t);
  console.log(layout);
  placeLayout(layout);
};

const handleSelect: Ontology.HandleSelect = ({
  selection,
  placeLayout,
  client,
  addStatus,
}) => {
  if (selection.length === 0) return;
  const key = selection[0].id.key;
  const name = selection[0].name;
  void (async () => {
    try {
      await retrieveAndPlaceLayout(client, key, placeLayout);
    } catch (e) {
      if (!(e instanceof Error)) throw e;
      addStatus({
        variant: "error",
        message: `Could not open ${name}`,
        description: e.message,
      });
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
          client.hardware.tasks.copy(id.key, `${name} (Snapshot)`, true),
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
  const hasNoSnapshots = resources.every((r) => r.data?.snapshot === false);
  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={onSelect}>
      <Group.GroupMenuItem selection={selection} />
      {hasNoSnapshots && <Range.SnapshotMenuItem key="snapshot" range={range} />}
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
  const layout = createLayout(task);
  placeLayout({ ...layout, tab: { mosaicKey: nodeKey, location } });
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
