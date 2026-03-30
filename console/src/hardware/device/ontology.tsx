// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/device/ontology.css";

import { device, type ontology } from "@synnaxlabs/client";
import { Access, Device, Flex, Menu, Text, Tree } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useMemo } from "react";

import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { CSS } from "@/css";
import { Group } from "@/group";
import { getContextMenuItems, getIcon, getMake } from "@/hardware/device/make";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";

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
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          shape={shape}
          rootID={rootID}
          onClick={() => group(props)}
        />
      )}
      {hasUpdatePermission && singleResource && <ContextMenu.RenameItem onClick={rename} />}
      {customMenuItems != null && (
        <>
          <Menu.Divider />
          {customMenuItems}
        </>
      )}
      {hasDeletePermission && (
        <>
          <Menu.Divider />
          <ContextMenu.DeleteItem onClick={handleDelete} />
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
          <Ontology.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
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
    </Tree.Item>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "device",
  icon,
  TreeContextMenu,
  Item,
};
