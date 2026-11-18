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
import { ContextMenu } from "@/platform/context-menu";
import { Credential } from "@/platform/credential";
import { CSS } from "@/platform/css";
import { Group } from "@/platform/group";
import { Tree } from "@/platform/tree";

const CreateArcIcon = Icon.createFromPath(
  "arc-create",
  "M9.66,14.34c-0.29,-0.29 -0.26,-0.64 -0.26,-0.64c-0.18,-0.18 -0.29,-0.43 -0.29,-0.71c0,-0.12 0.02,-0.24 0.06,-0.34c0.09,-0.38 0.28,-0.78 0.5,-1c0.29,-0.29 0.64,-0.26 0.64,-0.26c0.18,-0.18 0.43,-0.29 0.71,-0.29c0.12,0 0.24,0.02 0.34,0.06c0.38,0.09 0.78,0.28 1,0.5c0.29,0.29 0.26,0.64 0.26,0.64c0.18,0.18 0.29,0.43 0.29,0.71c0,0.12 -0.02,0.24 -0.06,0.34c-0.09,0.38 -0.28,0.78 -0.5,1c-0.29,0.29 -0.64,0.26 -0.64,0.26c-0.18,0.18 -0.43,0.29 -0.71,0.29c-0.12,0 -0.24,-0.02 -0.34,-0.06c-0.38,-0.09 -0.78,-0.28 -1,-0.5zM8.08,8.34c0.58,-0.36 1.62,-0.62 1.64,-0.62c0.15,-0.1 0.34,-0.15 0.53,-0.15c0.55,0 1,0.45 1,1c0,0.53 -0.42,0.97 -0.95,1c-0.39,0.13 -0.92,0.32 -1.16,0.47c-0.41,0.26 -0.71,0.52 -0.99,0.93c-0.02,0.03 -0.05,0.07 -0.07,0.1c-0.15,0.28 -0.34,0.65 -0.41,0.86c-0.11,0.35 -0.16,0.64 -0.17,1c-0.02,0.72 0.11,1.3 0.52,1.91c0.03,0.04 0.05,0.08 0.07,0.12c0.2,0.24 0.47,0.57 0.64,0.71c0.28,0.24 0.53,0.39 0.86,0.54c0.65,0.29 1.24,0.39 1.96,0.25c0.07,-0.02 0.15,-0.03 0.23,-0.03c0.55,0 1,0.45 1,1c0,0.51 -0.39,0.94 -0.88,0.99c-1.03,0.15 -2.2,0.01 -3.11,-0.39c-0.46,-0.2 -0.96,-0.52 -1.35,-0.85c-0.39,-0.33 -0.87,-1 -0.88,-1.01c-0.09,-0.08 -0.17,-0.17 -0.23,-0.28c-0.53,-0.9 -0.85,-2.03 -0.83,-3.02c0.01,-0.51 0.11,-1.09 0.27,-1.57c0.15,-0.48 0.58,-1.16 0.59,-1.19c0.04,-0.11 0.09,-0.21 0.16,-0.3c0.42,-0.57 0.99,-1.12 1.56,-1.47zM2.62,16.54c-0.44,-1.04 -0.63,-3.1 -0.64,-3.15c-0.05,-0.12 -0.08,-0.25 -0.08,-0.39c0,-0.55 0.45,-1 1,-1c0.55,0 1,0.45 1,1c0,0.02 0,0.05 0,0.07c0.03,0.37 0.08,1.03 0.15,1.36c0.1,0.48 0.23,0.89 0.41,1.33c0.37,0.88 0.86,1.59 1.52,2.26c1.33,1.33 2.97,2.11 4.98,2.08c0.01,0 0.03,0 0.04,0c0.02,0 0.05,0 0.07,0c0.35,-0.02 0.97,-0.07 1.29,-0.13c0.45,-0.09 0.85,-0.2 1.27,-0.37c0.84,-0.33 1.52,-0.77 2.18,-1.37c1.31,-1.2 2.13,-2.68 2.27,-4.54c0.03,-0.53 0.46,-0.95 1,-0.95c0.55,0 1,0.45 1,1c0,0.05 0,0.09 -0.01,0.13c-0.25,2.23 -1.35,4.4 -2.91,5.82c-0.79,0.72 -1.78,1.35 -2.79,1.75c-1,0.4 -2.94,0.56 -2.99,0.57c-0.11,0.05 -0.24,0.07 -0.37,0.07c-0.01,0 -0.03,0 -0.04,0c-2.43,-0.05 -4.81,-1.08 -6.39,-2.66c-0.8,-0.8 -1.5,-1.83 -1.95,-2.89zM13,7.5v-2h3.5v-3.5h2v3.5h3.5v2h-3.5v3.5h-2v-3.5z",
);

const ConnectDeviceIcon = Icon.createFromPath(
  "device-connect",
  "M5.8,19.6c-0.77,0 -1.4,-0.63 -1.4,-1.4v-0.1h-2.2c-0.22,0 -0.4,-0.18 -0.4,-0.4v-0.6c0,-0.22 0.18,-0.4 0.4,-0.4h2.2v-2h-2.2c-0.22,0 -0.4,-0.18 -0.4,-0.4v-0.6c0,-0.22 0.18,-0.4 0.4,-0.4h2.2v-2h-2.2c-0.22,0 -0.4,-0.18 -0.4,-0.4v-0.6c0,-0.22 0.18,-0.4 0.4,-0.4h2.2v-0.1c0,-0.77 0.63,-1.4 1.4,-1.4h0.1v-2.2c0,-0.22 0.18,-0.4 0.4,-0.4h0.6c0.22,0 0.4,0.18 0.4,0.4v2.2h2v-2.2c0,-0.22 0.18,-0.4 0.4,-0.4h0.55v0c0.25,0.03 0.45,0.24 0.45,0.5v2.1h0.05c0.28,0 0.5,0.22 0.5,0.5l0,0.6l2.75,0c0.28,0 0.5,0.22 0.5,0.5v2.75h0.6c0.28,0 0.5,0.22 0.5,0.5v0.05h2.1c0.26,0 0.47,0.2 0.5,0.45h0v0.55c0,0.22 -0.18,0.4 -0.4,0.4h-2.2v2h2.2c0.22,0 0.4,0.18 0.4,0.4v0.6c0,0.22 -0.18,0.4 -0.4,0.4h-2.2v0.1c0,0.77 -0.63,1.4 -1.4,1.4h-0.1v2.2c0,0.22 -0.18,0.4 -0.4,0.4h-0.6c-0.22,0 -0.4,-0.18 -0.4,-0.4v-2.2h-2v2.2c0,0.22 -0.18,0.4 -0.4,0.4h-0.6c-0.22,0 -0.4,-0.18 -0.4,-0.4v-2.2h-2v2.2c0,0.22 -0.18,0.4 -0.4,0.4h-0.6c-0.22,0 -0.4,-0.18 -0.4,-0.4v-2.2zM11.6,16.4c0.44,0 0.8,-0.36 0.8,-0.8v-3.2c0,-0.44 -0.36,-0.8 -0.8,-0.8h-3.2c-0.44,0 -0.8,0.36 -0.8,0.8v3.2c0,0.44 0.36,0.8 0.8,0.8zM13,7.75v-2.5h3.25v-3.25h2.5v3.25h3.25v2.5h-3.25v3.25h-2.5v-3.25z",
);

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
  typeName: "Driver",
  children: ({ client, id }) => [
    Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
    Tree.createSource(
      client.devices,
      { where: { rack: { eq: Number(id.key) }, grouped: false, nested: false } },
      Tree.toResource("device"),
    ),
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
