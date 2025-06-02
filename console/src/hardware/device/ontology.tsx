// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/device/ontology.css";

import { device, ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Menu as PMenu, Status, Text, Tooltip, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { Group } from "@/group";
import {
  CONFIGURE_LAYOUTS,
  getContextMenuItems,
  getIcon,
  getIconString,
  getMake,
  hasIdentifier,
  makeZ,
} from "@/hardware/device/make";
import { useState } from "@/hardware/device/Toolbar";
import { useRename } from "@/modals/Rename";
import { Ontology } from "@/ontology";

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) => {
    const device = await client.hardware.devices.retrieve(id.key);
    await client.hardware.devices.create({ ...device, name });
  },
};

const handleConfigure = ({
  selection: { resources },
  placeLayout,
  handleError,
}: Ontology.TreeContextMenuProps) => {
  const resource = resources[0];
  try {
    const make = makeZ.parse(resource.data?.make);
    placeLayout({ ...CONFIGURE_LAYOUTS[make], key: resource.id.key });
  } catch (e) {
    handleError(e, `Failed to configure ${resource.name}`);
  }
};

const useHandleChangeIdentifier = () => {
  const rename = useRename();
  return ({
    selection: { resources },
    handleError,
    client,
  }: Ontology.TreeContextMenuProps) => {
    const resource = resources[0];
    handleError(async () => {
      const device = await client.hardware.devices.retrieve(resource.id.key);
      const identifier =
        typeof device.properties.identifier === "string"
          ? device.properties.identifier
          : "";
      try {
        const newIdentifier = await rename(
          { initialValue: identifier, allowEmpty: false, label: "Identifier" },
          {
            icon: getIconString(getMake(resource.data?.make)),
            name: "Device.Identifier",
          },
        );
        if (newIdentifier == null) return;
        await client.hardware.devices.create({
          ...device,
          properties: { ...device.properties, identifier: newIdentifier },
        });
      } catch (e) {
        if (e instanceof Error && errors.Canceled.matches(e)) return;
        throw e;
      }
    }, "Failed to change identifier");
  };
};

const useDelete = () => {
  const confirm = Ontology.useConfirmDelete({ type: "Device" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ state: { nodes, setNodes }, selection: { resources } }) => {
      const prevNodes = Tree.deepCopy(nodes);
      if (!(await confirm(resources))) throw new errors.Canceled();
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
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (errors.Canceled.matches(e)) return;
      if (prevNodes != null) setNodes(prevNodes);
      handleError(e, `Failed to delete devices`);
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { nodes, resources },
  } = props;
  const singleResource = nodes.length === 1;
  const first = resources[0];
  const handleDelete = useDelete();
  const group = Group.useCreateFromSelection();
  const handleChangeIdentifier = useHandleChangeIdentifier();
  if (nodes.length === 0) return null;
  const handleSelect = {
    configure: () => handleConfigure(props),
    delete: () => handleDelete(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    group: () => group(props),
    changeIdentifier: () => handleChangeIdentifier(props),
  };
  const C = singleResource ? getContextMenuItems(resources[0].data?.make) : null;
  const customMenuItems = C ? <C {...props} /> : null;
  const showConfigure = singleResource && first.data?.configured !== true;
  const showChangeIdentifier =
    singleResource &&
    first.data?.configured === true &&
    hasIdentifier(getMake(first.data?.make));
  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Group.MenuItem selection={selection} />
      {singleResource && (
        <>
          <Menu.RenameItem />
          {(showConfigure || showChangeIdentifier) && <PMenu.Divider />}
          {showConfigure && (
            <PMenu.Item itemKey="configure" startIcon={<Icon.Hardware />}>
              Configure
            </PMenu.Item>
          )}
          {showChangeIdentifier && (
            <PMenu.Item itemKey="changeIdentifier" startIcon={<Icon.Hardware />}>
              Change Identifier
            </PMenu.Item>
          )}
        </>
      )}
      <PMenu.Divider />
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {customMenuItems != null && (
        <>
          <PMenu.Divider />
          {customMenuItems}
        </>
      )}
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const icon = (resource: ontology.Resource) => getIcon(getMake(resource.data?.make));

const Item: Tree.Item = ({ entry, className, ...rest }: Tree.ItemProps) => {
  const id = new ontology.ID(entry.key);
  const devState = useState(id.key);
  const variant = devState?.variant;
  let message = "Device State Unknown";
  if (
    devState?.details?.message != null &&
    typeof devState.details.message === "string"
  )
    message = devState.details.message;
  return (
    <Tree.DefaultItem
      className={CSS(className, CSS.B("device-ontology-item"))}
      entry={entry}
      {...rest}
    >
      {({ entry, onRename, key }) => (
        <>
          <Align.Space x grow align="center" className={CSS.B("name-location")}>
            <Text.MaybeEditable
              id={`text-${key}`}
              level="p"
              className={CSS.B("name")}
              allowDoubleClick={false}
              value={entry.name}
              disabled={!entry.allowRename}
              onChange={(name) => onRename?.(entry.key, name)}
            />
            <Text.Text level="small" shade={9} className={CSS.B("location")}>
              {entry.extraData?.location as string}
            </Text.Text>
          </Align.Space>
          <Tooltip.Dialog location="right">
            <Status.Text
              variant={variant ?? "error"}
              hideIcon
              level="small"
              weight={450}
            >
              {message}
            </Status.Text>
            <Status.Indicator variant={variant ?? "disabled"} />
          </Tooltip.Dialog>
        </>
      )}
    </Tree.DefaultItem>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: device.ONTOLOGY_TYPE,
  icon,
  hasChildren: false,
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
  Item,
};
