// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/device/tree.css";

import { device, type rack, status, type Synnax } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Access, Device } from "@synnaxlabs/pluto";

import { ContextMenuItems, getIcon, getIntegration } from "@/feature/device/make";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Tree } from "@/platform/tree";

const useDelete = Tree.createUseDelete({
  type: "Device",
  useUpdate: Device.useDelete,
  convertKey: String,
});

const useRename = Tree.createUseRename({
  useUpdate: Device.useRename,
  resourceID: device.resourceID,
  convertKey: String,
});

const retrieve: Tree.Retrieve = async (c, key) => await c.devices.retrieve({ key });

/** @returns the make that every selected device shares, once all of them load. */
const sharedMake = (
  devices: device.Device[] | undefined,
  count: number,
): string | null => {
  if (devices == null || devices.length !== count) return null;
  const makes = new Set(devices.map(({ make }) => make));
  return makes.size === 1 ? devices[0].make : null;
};

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID, parentID },
    state,
  } = props;
  const resourceIDs = ids.map((id) => device.resourceID(id.key));
  const hasUpdatePermission = Access.useUpdateGranted(resourceIDs);
  const hasDeletePermission = Access.useDeleteGranted(resourceIDs);
  const singleResource = ids.length === 1;
  const first = Tree.useResource(state, ids[0]);
  const { data: selected } = Device.useResultMultiple(
    ids.length === 0 ? null : { where: { key: { in: ids.map((id) => id.key) } } },
  );
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const handleLink = Core.useCopyLinkToClipboard();
  if (ids.length === 0) return null;
  const integration = getIntegration(sharedMake(selected, ids.length));
  const customMenuItems =
    integration == null ? null : (
      <ContextMenuItems {...props} integration={integration} />
    );
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && singleResource && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          shape={state.shape}
          rootID={rootID}
          parentID={parentID}
          onClick={() => group(props)}
        />
      )}
      {customMenuItems != null && (
        <>
          <Menu.Divider />
          {customMenuItems}
        </>
      )}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      {singleResource && (
        <>
          {first != null && (
            <Link.CopyContextMenuItem
              onClick={() =>
                handleLink({
                  name: first.name,
                  resourceID: device.resourceID(first.id.key),
                })
              }
            />
          )}
          <Tree.CopyPropertiesContextMenuItem {...props} retrieve={retrieve} />
        </>
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const Content = ({ resource, className, icon: _icon, ...rest }: Tree.ContentProps) => {
  const { itemKey } = rest;
  const query = { key: resource.id.key };
  const { data: devStatus } = Device.useResultStatus(query);
  const { data: make } = Device.useResultMake(query);
  const { data: location = "" } = Device.useResultLocation(query);
  const integration = getIntegration(make);
  const highlight = Tree.useHighlight();
  return (
    <PTree.Item className={CSS.cls(className, CSS.B("device-tree-item"))} {...rest}>
      <Flex.Box x grow align="center" className={CSS.B("name-location")}>
        <span className={CSS.B("device-icon")}>{getIcon(integration)}</span>
        <Text.MaybeEditable
          id={List.itemNameID(itemKey)}
          className={CSS.B("name")}
          allowDoubleClick={false}
          value={resource.name}
          onChange
          overflow="fade"
          status={status.keepVariants(devStatus?.variant, "disabled")}
          highlight={highlight}
        />
        <Tree.MatchGroup id={resource.id} />
        {location !== "" && (
          <Device.Location
            value={location}
            format={integration?.location}
            className={CSS.B("device-location")}
          />
        )}
      </Flex.Box>
      <Device.StatusIndicator status={devStatus} />
    </PTree.Item>
  );
};

/** Lists the devices that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "device",
  children: (client, key) =>
    Tree.createSource(
      client.devices,
      { where: { groups: [key] } },
      Tree.toResource("device"),
    ),
};

/** Lists the top level of the device tree: its groups and the racks in no group. */
export const listTreeRoot = (client: Synnax): Tree.Source[] => [
  Tree.createSource(
    client.groups,
    { scope: "device", nested: false },
    Tree.toResource("group"),
  ),
  Tree.createSource(client.racks, { where: { grouped: false } }, (r: rack.Rack) =>
    Tree.toResource("rack")(r.payload),
  ),
];

const children: Tree.Children = ({ client, id }) => [
  Tree.createSource(
    client.devices,
    { where: { parents: [id.key] } },
    Tree.toResource("device"),
  ),
];

const TreeItem = Tree.createItem({
  linkable: true,
  children,
  ContextMenu: TreeContextMenu,
  Content,
});

export const TREE_ITEMS = { device: TreeItem } satisfies Tree.Items;
