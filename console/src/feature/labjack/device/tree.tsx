// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { labjack, status } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Access, Device } from "@synnaxlabs/pluto";

import { TYPED_ADAPTER, TYPED_HARDWARE } from "@/feature/labjack/device/hardware";
import { useDelete, useRename, useRetrieve } from "@/feature/labjack/device/queries";
import { ContextMenu } from "@/platform/context-menu";
import { Device as PlatformDevice } from "@/platform/device";
import { Tree } from "@/platform/tree";

const useDeleteDevice = Tree.createUseDelete({
  type: "LabJack device",
  useUpdate: useDelete,
  convertKey: String,
});

const useRenameDevice = Tree.createUseRename({
  useUpdate: useRename,
  resourceID: labjack.device.resourceID,
  convertKey: String,
});

const ConfigureMenuItem = PlatformDevice.createConfigureMenuItem(TYPED_ADAPTER);

const ChangeIdentifierMenuItem = PlatformDevice.createChangeIdentifierMenuItem({
  adapter: TYPED_ADAPTER,
  icon: "Logo.LabJack",
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids },
  } = props;
  const resourceIDs = ids.map((id) => labjack.device.resourceID(id.key));
  const hasUpdatePermission = Access.useUpdateGranted(resourceIDs);
  const hasDeletePermission = Access.useDeleteGranted(resourceIDs);
  const handleDelete = useDeleteDevice(props);
  const rename = useRenameDevice(props);
  const configure = PlatformDevice.useConfigureModal();
  if (ids.length === 0) return null;
  return (
    <ContextMenu.Menu>
      {ids.length === 1 && (
        <>
          <ConfigureMenuItem
            {...props}
            onConfigure={(deviceKey) =>
              configure({ deviceKey, hardware: TYPED_HARDWARE })
            }
          />
          <ChangeIdentifierMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      {hasUpdatePermission && ids.length === 1 && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const Content = ({ resource, icon: _icon, ...rest }: Tree.ContentProps) => {
  const { itemKey } = rest;
  const { data: device } = useRetrieve({ key: resource.id.key });
  const highlight = Tree.useHighlight();
  return (
    <PTree.Item {...rest}>
      <Flex.Box x grow align="center">
        <Icon.Logo.LabJack />
        <Text.MaybeEditable
          id={List.itemNameID(itemKey)}
          allowDoubleClick={false}
          value={resource.name}
          onChange
          overflow="fade"
          status={status.keepVariants(device?.status?.variant, "disabled")}
          highlight={highlight}
        />
        <Tree.MatchGroup id={resource.id} />
      </Flex.Box>
      <Device.StatusIndicator status={device?.status} />
    </PTree.Item>
  );
};

const TreeItem = Tree.createItem({
  icon: <Icon.Logo.LabJack />,
  typeName: "LabJack device",
  Content,
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { labjack_device: TreeItem } satisfies Tree.Items;
