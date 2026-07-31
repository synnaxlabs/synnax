// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/device/tree.css";

import { device, type ontology, status } from "@synnaxlabs/client";
import {
  Access,
  Device,
  Flex,
  List,
  Menu,
  Text,
  Tree as PTree,
} from "@synnaxlabs/pluto";
import { useMemo } from "react";

import { getContextMenuItems, getIcon, getMake } from "@/feature/device/make";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Tree } from "@/platform/tree";

const useDelete = Tree.createUseDelete({
  type: "Device",
  query: Device.useDelete,
  convertKey: String,
});

const useRename = Tree.createUseRename({
  query: Device.useRename,
  ontologyID: device.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const ontologyIDs = useMemo(() => ids.map((id) => device.ontologyID(id.key)), [ids]);
  const hasUpdatePermission = Access.useUpdateGranted(ontologyIDs);
  const hasDeletePermission = Access.useDeleteGranted(ontologyIDs);
  const singleResource = ids.length === 1;
  const first = getResource(ids[0]);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const handleLink = Cluster.useCopyLinkToClipboard();
  if (ids.length === 0) return null;
  const C = singleResource ? getContextMenuItems(first.data?.make) : null;
  const customMenuItems = C ? <C {...props} /> : null;
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && singleResource && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          shape={shape}
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
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({
                name: first.name,
                ontologyID: device.ontologyID(first.id.key),
              })
            }
          />
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
  const devStatus = Device.useRetrieve({ key: resource.id.key }).data?.status;
  return (
    <PTree.Item className={CSS(className, CSS.B("device-ontology-item"))} {...rest}>
      <Flex.Box x grow align="center" className={CSS.B("name-location")}>
        {icon(resource)}
        <Text.MaybeEditable
          id={List.itemNameID(itemKey)}
          className={CSS.B("name")}
          allowDoubleClick={false}
          value={resource.name}
          onChange
          overflow="ellipsis"
          status={status.keepVariants(devStatus?.variant, "disabled")}
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
    </PTree.Item>
  );
};

const TreeItem = Tree.createItem({
  type: "device",
  icon,
  ContextMenu: TreeContextMenu,
  Content,
});

export const TREE_ITEMS = { device: TreeItem } satisfies Tree.Items;
