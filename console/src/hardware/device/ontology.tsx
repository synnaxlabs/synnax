// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { type Make, makeZ } from "@/hardware/device/make";
import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

const ZERO_CONFIGURE_LAYOUTS: Record<Make, Layout.BaseState> = {
  [LabJack.Device.MAKE]: LabJack.Device.CONFIGURE_LAYOUT,
  [NI.Device.MAKE]: NI.Device.CONFIGURE_LAYOUT,
  [OPC.Device.MAKE]: OPC.Device.CONFIGURE_LAYOUT,
};

const CONTEXT_MENUS_ITEMS: Record<
  Make,
  (props: Ontology.TreeContextMenuProps) => ReactElement | null
> = {
  [LabJack.Device.MAKE]: LabJack.DeviceServices.ContextMenuItems,
  [NI.Device.MAKE]: NI.DeviceServices.ContextMenuItems,
  [OPC.Device.MAKE]: OPC.DeviceServices.ContextMenuItems,
};

const handleConfigure = ({
  selection: { resources },
  placeLayout,
  handleException,
}: Ontology.TreeContextMenuProps): void => {
  const resource = resources[0];
  try {
    const make = makeZ.parse(resource.data?.make);
    const baseLayout = ZERO_CONFIGURE_LAYOUTS[make];
    placeLayout({ ...baseLayout, key: resource.id.key });
  } catch (e) {
    handleException(e, `Failed to configure ${resource.name}`);
  }
};

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = Ontology.useConfirmDelete({ type: "Device" });
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
    onError: (e, { handleException, state: { setNodes } }, prevNodes) => {
      if (errors.CANCELED.matches(e)) return;
      if (prevNodes != null) setNodes(prevNodes);
      handleException(e, `Failed to delete devices`);
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
  const make = makeZ.safeParse(resources[0].data?.make)?.data;
  const C = make != null ? CONTEXT_MENUS_ITEMS[make] : null;
  const customMenuItems = C != null ? <C {...props} /> : null;
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
      {customMenuItems != null && <PMenu.Divider />}
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
  ...Ontology.BASE_SERVICE,
  type: device.ONTOLOGY_TYPE,
  icon: <Icon.Device />,
  hasChildren: false,
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
};
