// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/rack/tree.css";

import { arc, device, group, rack, type resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Access, Rack } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Arc } from "@/feature/arc";
import { Device } from "@/feature/device";
import { NI } from "@/feature/ni";
import { FLAGS } from "@/flags";
import { ContextMenu } from "@/platform/context-menu";
import { Credential } from "@/platform/credential";
import { CSS } from "@/platform/css";
import { Group } from "@/platform/group";
import { Tree } from "@/platform/tree";

const CreateArcIcon = Icon.createComposite(Icon.Arc, {
  topRight: Icon.Add,
});

const ConnectDeviceIcon = Icon.createComposite(Icon.Device, {
  topRight: Icon.Add,
});

const useRename = Tree.createUseRename({
  useUpdate: Rack.useRename,
  resourceID: rack.resourceID,
  convertKey: Number,
});

const Content = ({ resource, icon: _icon, ...rest }: Tree.ContentProps) => {
  const { itemKey } = rest;
  const status = Rack.useResult({ key: Number(resource.id.key) }).data?.status;
  const highlight = Tree.useHighlight();
  return (
    <PTree.Item {...rest}>
      <Icon.Rack />
      <Text.MaybeEditable
        id={List.itemNameID(itemKey)}
        allowDoubleClick={false}
        value={resource.name}
        overflow="fade"
        className={CSS.B("rack-tree-name")}
        grow
        onChange
        highlight={highlight}
      />
      <Tree.MatchGroup id={resource.id} />
      <Rack.StatusIndicator status={status} />
    </PTree.Item>
  );
};

const useDelete = Tree.createUseDelete({
  type: "Rack",
  useUpdate: Rack.useDelete,
  convertKey: Number,
});

interface IntegrationItemsProps {
  rackKey: rack.Key;
}

/** The menu items of each integration that the rack's Driver runs. */
const IntegrationItems = ({ rackKey }: IntegrationItemsProps): ReactElement => {
  Rack.useEnsure({ key: rackKey });
  const keys = Rack.useIntegrations({ key: rackKey });
  return (
    <>
      {Device.INTEGRATIONS.map(({ key, RackMenuItems }) =>
        RackMenuItems != null && keys.includes(key) ? (
          <RackMenuItems key={key} rackKey={rackKey} />
        ) : null,
      )}
    </>
  );
};

const retrieve: Tree.Retrieve = async (c, key) =>
  (await c.racks.retrieve({ key: Number(key) })).payload;

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection,
    state: { shape, getResource },
  } = props;
  const { ids, rootID, parentID } = selection;
  const resourceIDs = ids.map((id) => rack.resourceID(Number(id.key)));
  const hasUpdatePermission = Access.useUpdateGranted(resourceIDs);
  const hasArcCreatePermission = Access.useCreateGranted(arc.TYPE_RESOURCE_ID);
  const hasDeletePermission = Access.useDeleteGranted(resourceIDs);
  const hasDeviceCreatePermission = Access.useCreateGranted(device.TYPE_RESOURCE_ID);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const groupFromSelection = Group.useCreateFromSelection();
  const create = Arc.useCreate();
  const openConnectDevice = Device.useConnectModal();
  const openCredentials = Credential.useModal();
  const isSingle = ids.length === 1;
  const rackKey = Number(ids[0]?.key ?? 0);
  const handleCreate = () => create();
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && isSingle && <ContextMenu.RenameItem onClick={rename} />}
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          rootID={rootID}
          parentID={parentID}
          shape={shape}
          onClick={() => groupFromSelection(props)}
        />
      )}
      <Menu.Divider />
      {hasDeviceCreatePermission && isSingle && (
        <Menu.Item
          itemKey="connectDevice"
          onClick={() => openConnectDevice({ rack: rackKey })}
        >
          <ConnectDeviceIcon />
          Connect device
        </Menu.Item>
      )}
      {hasUpdatePermission && isSingle && (
        <>
          {hasArcCreatePermission && (
            <Menu.Item itemKey="createArc" onClick={handleCreate}>
              <CreateArcIcon />
              Create Arc automation
            </Menu.Item>
          )}
          <IntegrationItems rackKey={rackKey} />
          <Menu.Item
            itemKey="signIn"
            onClick={() =>
              openCredentials({
                subject: resourceIDs[0],
                title: ["Sign-in", getResource(ids[0]).name],
              })
            }
          >
            <Icon.Lock />
            Manage sign-in
          </Menu.Item>
        </>
      )}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      {isSingle && (
        <Tree.CopyPropertiesContextMenuItem {...props} retrieve={retrieve} />
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const toResource = (r: rack.Rack): Tree.Resource => Tree.toResource("rack")(r.payload);

/** Lists the racks that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "rack",
  children: (client, key) =>
    Tree.createSource(client.racks, { where: { groups: [key] } }, toResource),
};

const DROPPABLE_TYPES = new Set<resource.Type>(["group", "device"]);

const TreeItem = Tree.createItem({
  icon: <Icon.Rack />,
  children: ({ client, id }) => [
    Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
    Tree.createSource(
      client.devices,
      { where: { rack: { eq: Number(id.key) }, grouped: false, nested: false } },
      Tree.toResource("device"),
    ),
    ...(FLAGS.typedNIDevices
      ? [NI.Chassis.createRackSource(client, Number(id.key))]
      : []),
  ],
  canDrop: (props) =>
    props.moved.every(({ id }) => DROPPABLE_TYPES.has(id.type)) &&
    group.canMove({ ...props, destination: props.destination.id }),
  drop: async ({ client, destination, items }) => {
    await client.batch.send(
      client.groups.moveRequests(
        items.map(({ id }) => id),
        destination.id,
      ),
    );
  },
  ContextMenu: TreeContextMenu,
  Content,
});

export const TREE_ITEMS = { rack: TreeItem } satisfies Tree.Items;
