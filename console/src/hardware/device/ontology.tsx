// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/device/ontology.css";

import { ontology } from "@synnaxlabs/client";
import { Device, Flex, Icon, Menu as PMenu, Text, Tree } from "@synnaxlabs/pluto";
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
import { useRename } from "@/modals/Rename";
import { Ontology } from "@/ontology";

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) => {
    const device = await client.hardware.devices.retrieve({ key: id.key });
    await client.hardware.devices.create({ ...device, name });
  },
};

const handleConfigure = ({
  selection: { resourceIDs },
  state: { getResource },
  placeLayout,
  handleError,
}: Ontology.TreeContextMenuProps) => {
  const resource = getResource(resourceIDs[0]);
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
    selection: { resourceIDs },
    state: { getResource },
    handleError,
    client,
  }: Ontology.TreeContextMenuProps) => {
    const resource = getResource(resourceIDs[0]);
    handleError(async () => {
      const device = await client.hardware.devices.retrieve({ key: resource.id.key });
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
    onMutate: async ({
      state: { nodes, setNodes },
      selection: { resourceIDs },
      state: { getResource },
    }) => {
      const prevNodes = Tree.deepCopy(nodes);
      const resources = getResource(resourceIDs);
      if (!(await confirm(resources))) throw new errors.Canceled();
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => ontology.idToString(id)),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ selection: { resourceIDs }, client }) =>
      await client.hardware.devices.delete(resourceIDs.map((id) => id.key)),
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (errors.Canceled.matches(e)) return;
      if (prevNodes != null) setNodes(prevNodes);
      handleError(e, `Failed to delete devices`);
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resourceIDs, rootID },
    state: { getResource, shape },
  } = props;
  const singleResource = resourceIDs.length === 1;
  const first = getResource(resourceIDs[0]);
  const handleDelete = useDelete();
  const group = Group.useCreateFromSelection();
  const handleChangeIdentifier = useHandleChangeIdentifier();
  if (resourceIDs.length === 0) return null;
  const handleSelect = {
    configure: () => handleConfigure(props),
    delete: () => handleDelete(props),
    rename: () => Text.edit(ontology.idToString(resourceIDs[0])),
    group: () => group(props),
    changeIdentifier: () => handleChangeIdentifier(props),
  };
  const C = singleResource ? getContextMenuItems(first.data?.make) : null;
  const customMenuItems = C ? <C {...props} /> : null;
  const showConfigure = singleResource && first.data?.configured !== true;
  const showChangeIdentifier =
    singleResource &&
    first.data?.configured === true &&
    hasIdentifier(getMake(first.data?.make));
  return (
    <PMenu.Menu onChange={handleSelect} level="small" gap="small">
      <Group.MenuItem resourceIDs={resourceIDs} shape={shape} rootID={rootID} />
      {singleResource && (
        <>
          <Menu.RenameItem />
          {(showConfigure || showChangeIdentifier) && <PMenu.Divider />}
          {showConfigure && (
            <PMenu.Item itemKey="configure">
              <Icon.Hardware />
              Configure
            </PMenu.Item>
          )}
          {showChangeIdentifier && (
            <PMenu.Item itemKey="changeIdentifier">
              <Icon.Hardware />
              Change Identifier
            </PMenu.Item>
          )}
        </>
      )}
      <PMenu.Divider />
      <PMenu.Item itemKey="delete">
        <Icon.Delete />
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

const Item = ({
  id,
  resource,
  className,
  onRename,
  ...rest
}: Ontology.TreeItemProps) => {
  const { itemKey } = rest;
  const devStatus = Device.useRetrieve({ key: id.key }).data?.status;
  return (
    <Tree.Item className={CSS(className, CSS.B("device-ontology-item"))} {...rest}>
      <Flex.Box x grow align="center" className={CSS.B("name-location")}>
        {icon(resource)}
        <Text.MaybeEditable
          id={itemKey}
          className={CSS.B("name")}
          allowDoubleClick={false}
          value={resource.name}
          onChange={onRename}
          overflow="ellipsis"
        />
        <Text.Text
          level="small"
          color={9}
          className={CSS.B("location")}
          overflow="nowrap"
        >
          {typeof resource.data?.location === "string" ? resource.data.location : ""}
        </Text.Text>
      </Flex.Box>
      <Device.StatusIndicator status={devStatus} />
    </Tree.Item>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "device",
  icon,
  hasChildren: false,
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
  Item,
};
