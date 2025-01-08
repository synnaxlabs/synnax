// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Status, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { LabJack } from "@/hardware/labjack";
import { type Make, makeZ } from "@/hardware/makes";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { type Layout } from "@/layout";
import { type Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";

type DeviceLayoutCreator = (
  device: string,
  initial: Partial<Layout.State>,
) => Layout.Creator;

const ZERO_LAYOUT_STATES: Record<Make, DeviceLayoutCreator> = {
  [LabJack.MAKE]: LabJack.Device.createConfigureLayout,
  [NI.MAKE]: NI.Device.createConfigureLayout,
  [OPC.MAKE]: OPC.Device.createConfigureLayout,
};

const CONTEXT_MENUS: Record<
  Make,
  (props: Ontology.TreeContextMenuProps) => ReactElement | null
> = {
  [LabJack.MAKE]: LabJack.Device.ContextMenuItems,
  [NI.MAKE]: NI.Device.ContextMenuItems,
  [OPC.MAKE]: OPC.Device.ContextMenuItems,
};

const handleSelect: Ontology.HandleSelect = () => {};

const handleConfigure = ({
  selection: { resources },
  placeLayout,
  addStatus,
}: Ontology.TreeContextMenuProps): void => {
  const resource = resources[0];
  try {
    const make = makeZ.parse(resource.data?.make);
    const baseLayout = ZERO_LAYOUT_STATES[make];
    const key = resource.id.key;
    placeLayout(baseLayout(key, {}));
  } catch (e) {
    Status.handleException(e, `Failed to configure ${resource.name}`, addStatus);
  }
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
      Status.handleException(e, `Failed to delete devices`, addStatus);
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
    const C = CONTEXT_MENUS[make as Make];
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
