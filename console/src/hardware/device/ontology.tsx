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
import { Device, Flex, Icon, Menu as PMenu, Text, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

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
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";

const handleConfigure = ({
  selection: { ids },
  state: { getResource },
  placeLayout,
  handleError,
}: Ontology.TreeContextMenuProps) => {
  const resource = getResource(ids[0]);
  try {
    const make = makeZ.parse(resource.data?.make);
    placeLayout({ ...CONFIGURE_LAYOUTS[make], key: resource.id.key });
  } catch (e) {
    handleError(e, `Failed to configure ${resource.name}`);
  }
};

const useHandleChangeIdentifier = () => {
  const rename = Modals.useRename();
  return ({
    selection: { ids },
    state: { getResource },
    handleError,
    client,
  }: Ontology.TreeContextMenuProps) => {
    const resource = getResource(ids[0]);
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

const useDelete = ({
  state: { getResource },
  selection: { ids },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const confirm = Ontology.useConfirmDelete({ type: "Device" });
  const keys = useMemo(() => ids.map((id) => id.key), [ids]);
  const beforeUpdate = useCallback(
    async () => await confirm(getResource(ids)),
    [confirm, getResource, ids],
  );
  const { update } = Device.useDelete({ beforeUpdate });
  return useCallback(() => update(keys), [update, keys]);
};

const useRename = (props: Ontology.TreeContextMenuProps) => {
  const { update } = Device.useRename({
    beforeUpdate: async ({ value }) => {
      const [name, renamed] = await Text.asyncEdit(
        ontology.idToString(device.ontologyID(value.key)),
      );
      if (!renamed) return false;
      return { ...value, name };
    },
  });
  const firstId = props.selection.ids[0];
  return useCallback(() => update({ key: firstId.key, name: "" }), []);
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const singleResource = ids.length === 1;
  const first = getResource(ids[0]);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const handleChangeIdentifier = useHandleChangeIdentifier();
  if (ids.length === 0) return null;
  const handleSelect = {
    configure: () => handleConfigure(props),
    delete: handleDelete,
    rename,
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
      <Group.MenuItem ids={ids} shape={shape} rootID={rootID} />
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
  TreeContextMenu,
  Item,
};
