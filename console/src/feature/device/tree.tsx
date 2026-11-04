// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/device/tree.css";

import { device, type ontology, type rack, resource, status } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Access, Device } from "@synnaxlabs/pluto";

import { ContextMenuItems, getIcon, getMake } from "@/feature/device/make";
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

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state,
  } = props;
  const resourceIDs = ids.map((id) => device.resourceID(id.key));
  const hasUpdatePermission = Access.useUpdateGranted(resourceIDs);
  const hasDeletePermission = Access.useDeleteGranted(resourceIDs);
  const singleResource = ids.length === 1;
  const first = Tree.useResource(state, ids[0]);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const handleLink = Core.useCopyLinkToClipboard();
  if (ids.length === 0) return null;
  const make = singleResource ? getMake(first?.data?.make) : null;
  const customMenuItems =
    make == null ? null : <ContextMenuItems {...props} make={make} />;
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
          <Tree.CopyPropertiesContextMenuItem {...props} />
        </>
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const icon = (resource: ontology.Resource) => getIcon(getMake(resource.data?.make));

const Content = ({ resource, className, icon: _icon, ...rest }: Tree.ContentProps) => {
  const { itemKey } = rest;
  const devStatus = Device.useResultStatus({ key: resource.id.key }).data;
  const location =
    typeof resource.data?.location === "string" ? resource.data.location : "";
  return (
    <PTree.Item className={CSS.cls(className, CSS.B("device-ontology-item"))} {...rest}>
      <Flex.Box x grow align="center" className={CSS.B("name-location")}>
        {icon(resource)}
        <Text.MaybeEditable
          id={List.itemNameID(itemKey)}
          className={CSS.B("name")}
          allowDoubleClick={false}
          value={resource.name}
          onChange
          overflow="fade"
          status={status.keepVariants(devStatus?.variant, "disabled")}
        />
        {location !== "" && (
          <Text.Text
            level="small"
            color={9}
            className={CSS.B("location")}
            overflow="fade"
          >
            {location}
          </Text.Text>
        )}
      </Flex.Box>
      <Device.StatusIndicator status={devStatus} />
    </PTree.Item>
  );
};

// A rack carries a task client, so the tree keeps only its fields.
export const toRackResource = ({
  key,
  name,
  integrations,
  embedded,
  group,
}: rack.Rack): ontology.Resource =>
  Tree.toResource("rack")({ key, name, integrations, embedded, group });

/** Lists the devices that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "device",
  children: (client, key) =>
    Tree.createSource(client.devices, { groups: [key] }, Tree.toResource("device")),
};

const children: Tree.Children = ({ client, id }) =>
  resource.idsEqual(id, device.TYPE_RESOURCE_ID)
    ? [
        Tree.createSource(
          client.groups,
          { scopes: ["device"], nested: false },
          Tree.toResource("group"),
        ),
        Tree.createSource(client.racks, { grouped: false }, toRackResource),
      ]
    : // A device lists its chassis modules through the ontology until PR 5.
      [Tree.createOntologySource(client, id)];

const TreeItem = Tree.createItem({
  type: "device",
  icon,
  children,
  ContextMenu: TreeContextMenu,
  Content,
});

export const TREE_ITEMS = { device: TreeItem } satisfies Tree.Items;
