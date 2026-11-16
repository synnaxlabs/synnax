// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource, schematic } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, type Flux, Mosaic, Schematic as Base } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";

import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useDelete = Tree.createUseDelete({
  type: "Schematic",
  useUpdate: Base.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, store }) => {
    store.dispatch(Session.Schematic.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useDuplicate = (props: Tree.ContextMenuProps): (() => void) => {
  const {
    selection: { ids },
    state: { getResource },
  } = props;
  const rename = Base.useRename();
  const copy = Base.useCopy({
    afterSuccess: async ({ data }: Flux.AfterSuccessParams<schematic.Schematic>) => {
      const id = schematic.resourceID(data.key);
      const [name, renamed] = await Text.asyncEdit(
        List.itemNameID(resource.idToString(id)),
      );
      if (!renamed) return;
      await rename.updateAsync({ key: data.key, name });
    },
  });
  return () =>
    ids.map((id) => {
      const name = `${getResource(id).name} (copy)`;
      copy.update({ key: id.key, name });
    });
};

const useRename = Tree.createUseRename({
  useUpdate: Base.useRename,
  resourceID: schematic.resourceID,
  convertKey: String,
});

const retrieve: Tree.Retrieve = async (c, key) => await c.schematics.retrieve(key);

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID, parentID },
    state,
  } = props;
  const hasCreatePermission = Access.useCreateGranted(schematic.TYPE_RESOURCE_ID);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const handleDelete = useDelete(props);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const handleDuplicate = useDuplicate(props);
  const handleExport = Export.useResource();
  const handleLink = Core.useCopyLinkToClipboard();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const firstID = ids[0];
  const resources = Tree.useResources(state, ids);
  const first = resources[0];
  const singleResource = ids.length === 1;
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          {singleResource && <ContextMenu.RenameItem onClick={rename} />}
          <Group.ContextMenuItem
            ids={ids}
            shape={state.shape}
            rootID={rootID}
            parentID={parentID}
            onClick={() => group(props)}
          />
        </>
      )}
      <Menu.Divider />
      {hasCreatePermission && (
        <Menu.Item itemKey="duplicate" onClick={handleDuplicate}>
          <Icon.Copy />
          Duplicate
        </Menu.Item>
      )}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      {singleResource && (
        <>
          <Export.ContextMenuItem onClick={() => handleExport(first.id)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, resourceID: firstID })}
          />
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

/** Lists the schematics that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "schematic",
  children: (client, key) =>
    Tree.createSource(
      client.schematics,
      { groups: [key] },
      Tree.toResource("schematic"),
    ),
};

const TreeItem = Tree.createItem({
  linkable: true,
  icon: <Icon.Schematic />,
  useOnSelect: Panel.useOpenResource,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(resource.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { schematic: TreeItem } satisfies Tree.Items;
