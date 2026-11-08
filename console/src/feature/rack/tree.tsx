// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/rack/tree.css";

import {
  arc,
  device,
  type ontology,
  rack,
  type resource,
  task,
} from "@synnaxlabs/client";
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
import { OPCUA } from "@/feature/opcua";
import { ContextMenu } from "@/platform/context-menu";
import { Credential } from "@/platform/credential";
import { CSS } from "@/platform/css";
import { Group } from "@/platform/group";
import { Tree } from "@/platform/tree";

const NI_INTEGRATION_NAME = "ni";

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
  const { data } = Rack.useResult({ key: Number(resource.id.key) });

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
      />
      {data?.version != null && (
        <Text.Text level="small" color={9}>
          v{data.version}
        </Text.Text>
      )}
      <Rack.StatusIndicator status={data?.status} />
    </PTree.Item>
  );
};

const useDelete = Tree.createUseDelete({
  type: "Rack",
  useUpdate: Rack.useDelete,
  convertKey: Number,
});

interface NIScannerItemProps {
  rackKey: rack.Key;
}

const NIScannerItem = ({ rackKey }: NIScannerItemProps): ReactElement | null => {
  Rack.useEnsure({ key: rackKey });
  const integrations = Rack.useIntegrations({ key: rackKey });
  const toggleScanner = NI.Task.useToggleScanner(rackKey);
  if (!integrations.includes(NI_INTEGRATION_NAME)) return null;
  return (
    <Menu.Item itemKey="toggleNIScanner" onClick={toggleScanner}>
      <Icon.Logo.NI />
      Toggle NI device scanner
    </Menu.Item>
  );
};

interface OPCUADriverCertificateItemProps {
  rackKey: rack.Key;
}

const OPCUADriverCertificateItem = ({
  rackKey,
}: OPCUADriverCertificateItemProps): ReactElement | null => {
  Rack.useEnsure({ key: rackKey });
  const integrations = Rack.useIntegrations({ key: rackKey });
  const openCertificate = OPCUA.Device.useDriverCertificateModal();
  if (!integrations.includes(OPCUA.Device.INTEGRATION)) return null;
  return (
    <Menu.Item
      itemKey="showOPCUADriverCertificate"
      onClick={() => void openCertificate({ rackKey })}
    >
      <Icon.Logo.OPCUA />
      Show OPC UA Driver certificate
    </Menu.Item>
  );
};

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
  const hasTaskCreatePermission = Access.useCreateGranted(task.TYPE_RESOURCE_ID);
  const hasDeviceCreatePermission = Access.useCreateGranted(device.TYPE_RESOURCE_ID);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
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
          onClick={() => group(props)}
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
          {hasTaskCreatePermission && <NIScannerItem rackKey={rackKey} />}
          <OPCUADriverCertificateItem rackKey={rackKey} />
          <Menu.Item
            itemKey="credentials"
            onClick={() =>
              openCredentials({
                subject: resourceIDs[0],
                title: ["Credentials", getResource(ids[0]).name],
              })
            }
          >
            <Icon.Lock />
            Manage credentials
          </Menu.Item>
        </>
      )}
      <Menu.Divider />
      {isSingle && <Tree.CopyPropertiesContextMenuItem {...props} />}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const toResource = (r: rack.Rack): ontology.Resource =>
  Tree.toResource("rack")(r.payload);

/** Lists the racks that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "rack",
  children: (client, key) =>
    Tree.createSource(client.racks, { where: { groups: [key] } }, toResource),
};

const DROPPABLE_TYPES = new Set<resource.Type>(["group", "device"]);

const TreeItem = Tree.createItem({
  type: "rack",
  icon: <Icon.Rack />,
  children: ({ client, id }) => [
    Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
    Tree.createSource(
      client.devices,
      { where: { rack: { eq: Number(id.key) }, grouped: false, nested: false } },
      Tree.toResource("device"),
    ),
  ],
  canDrop: (props) =>
    props.moved.every(({ id }) => DROPPABLE_TYPES.has(id.type)) && Group.canMove(props),
  drop: async ({ client, destination, items }) => {
    await client.batch.send(
      Group.moveRequests(
        client,
        items.map(({ id }) => id),
        destination.id,
      ),
    );
  },
  ContextMenu: TreeContextMenu,
  Content,
});

export const TREE_ITEMS = { rack: TreeItem } satisfies Tree.Items;
