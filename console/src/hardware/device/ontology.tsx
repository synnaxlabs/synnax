// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";

type DeviceLayoutCreator = (
  device: string,
  initial: Partial<Layout.State>,
) => Layout.Creator;
const ZERO_LAYOUT_STATES: Record<string, DeviceLayoutCreator> = {
  [LabJack.Device.CONFIGURE_LAYOUT_TYPE]: LabJack.Device.createConfigureLayout,
  [NI.Device.CONFIGURE_LAYOUT_TYPE]: NI.Device.createConfigureLayout,
  [OPC.Device.CONFIGURE_LAYOUT_TYPE]: OPC.Device.createConfigureLayout,
};

const CONTEXT_MENUS: Record<
  string,
  (props: Ontology.TreeContextMenuProps) => ReactElement | null
> = {
  [LabJack.MAKE]: LabJack.Device.ContextMenuItems,
  [NI.MAKE]: NI.Device.ContextMenuItems,
  [OPC.MAKE]: OPC.Device.ContextMenuItems,
};

export const handleSelect: Ontology.HandleSelect = () => {};

const handleConfigure = ({
  selection,
  client,
  placeLayout,
}: Ontology.TreeContextMenuProps): void => {
  if (selection.nodes.length === 0) return;
  const first = selection.resources[0];
  void (async () => {
    const d = await client.hardware.devices.retrieve(first.id.key);
    const baseLayout = ZERO_LAYOUT_STATES[`configure_${d.make}`];
    if (baseLayout == null) return;
    return placeLayout(baseLayout(d.key, {}));
  })();
};

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "Device" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
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
    mutationFn: async ({ selection, client }) =>
      await client.hardware.devices.delete(selection.resources.map((r) => r.id.key)),
    onError: (e, { addStatus, state: { setNodes } }, prevNodes) => {
      if (errors.CANCELED.matches(e)) return;
      if (prevNodes != null) setNodes(prevNodes);
      addStatus({
        variant: "error",
        message: `Failed to delete devices`,
        description: e.message,
      });
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { nodes, resources },
  } = props;
  if (nodes.length === 0) return null;
  const singleResource = nodes.length === 1;
  const first = resources[0];
  const del = useDelete();
  const group = Group.useCreateFromSelection();
  const handleSelect = {
    configure: () => handleConfigure(props),
    delete: () => del(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    group: () => group(props),
  };
  const make = resources[0].data?.make;
  let customMenuItems: ReactElement | null = null;
  if (make != null) {
    const C = CONTEXT_MENUS[make as string];
    if (C != null) customMenuItems = <C {...props} />;
  }
  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Group.GroupMenuItem selection={selection} />
      {singleResource && (
        <>
          <Menu.RenameItem />
          {first.data?.configured !== true && (
            <>
              <PMenu.Divider />
              <PMenu.Item itemKey="configure" startIcon={<Icon.Hardware />}>
                Configure
              </PMenu.Item>
            </>
          )}
        </>
      )}
      <PMenu.Divider />
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {customMenuItems}
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) => {
    const device = await client.hardware.devices.retrieve(id.key);
    await client.hardware.devices.create({ ...device, name });
  },
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "device",
  hasChildren: false,
  icon: <Icon.Device />,
  canDrop: () => false,
  onSelect: handleSelect,
  TreeContextMenu,
  haulItems: () => [],
  allowRename: () => true,
  onRename: handleRename,
};
