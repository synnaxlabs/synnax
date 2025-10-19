// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/device/ontology.css";

import { device, type ontology } from "@synnaxlabs/client";
import {
  ContextMenu as PContextMenu,
  Device,
  Flex,
  Icon,
  Text,
  Tree,
} from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";

import { ContextMenu } from "@/components";
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
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";

const configure = ({
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

const useDelete = createUseDelete({
  type: "Device",
  query: Device.useDelete,
  convertKey: String,
});

const useRename = createUseRename({
  query: Device.useRename,
  ontologyID: device.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids },
    state: { getResource },
  } = props;
  const singleResource = ids.length === 1;
  const first = getResource(ids[0]);
  const handleDelete = useDelete(props);
  const handleRename = useRename(props);
  const changeIdentifier = useHandleChangeIdentifier();
  if (ids.length === 0) return null;
  const handleConfigure = () => configure(props);
  const handleChangeIdentifier = () => changeIdentifier(props);
  const C = singleResource ? getContextMenuItems(first.data?.make) : null;
  const customMenuItems = C ? <C {...props} /> : null;
  const showConfigure = singleResource && first.data?.configured !== true;
  const showChangeIdentifier =
    singleResource &&
    first.data?.configured === true &&
    hasIdentifier(getMake(first.data?.make));
  return (
    <>
      {singleResource && (
        <>
          <ContextMenu.RenameItem onClick={handleRename} showBottomDivider />
          {showConfigure && (
            <PContextMenu.Item onClick={handleConfigure}>
              <Icon.Hardware />
              Configure
            </PContextMenu.Item>
          )}
          {showChangeIdentifier && (
            <PContextMenu.Item onClick={handleChangeIdentifier}>
              <Icon.Hardware />
              Change identifier
            </PContextMenu.Item>
          )}
          {(showConfigure || showChangeIdentifier) && <PContextMenu.Divider />}
        </>
      )}
      <Group.ContextMenuItem {...props} showBottomDivider />
      <ContextMenu.DeleteItem onClick={handleDelete} showBottomDivider />
      {customMenuItems != null && (
        <>
          {customMenuItems}
          <PContextMenu.Divider />
        </>
      )}
      {singleResource && <Ontology.CopyContextMenuItem {...props} showBottomDivider />}
      <ContextMenu.ReloadConsoleItem />
    </>
  );
};

const icon = (resource: ontology.Resource) => getIcon(getMake(resource.data?.make));

const Item = ({ id, resource, className, ...rest }: Ontology.TreeItemProps) => {
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
          onChange
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
