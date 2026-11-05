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
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, type Flux, Mosaic, Schematic as Base } from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";

import { Symbol } from "@/feature/schematic/symbol";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
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

const useCopy = (props: Tree.ContextMenuProps): (() => void) => {
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
      copy.update({ key: id.key, name, snapshot: false });
    });
};

const useRangeSnapshot = () => {
  const addStatus = Status.useAdder();
  const rng = Range.useResolve();
  const { update } = Base.useSnapshot({
    afterSuccess: ({
      data: { schematics },
    }: Flux.AfterSuccessParams<Base.SnapshotParams>) => {
      const names = array.toArray(schematics).map((s) => s.name);
      const joined = strings.naturalLanguageJoin(names);
      const single = names.length === 1;
      addStatus({
        variant: "success",
        name: rng?.name ?? "",
        message: single
          ? "Snapshotted schematic"
          : `Snapshotted ${names.length} schematics`,
        description: single
          ? `Created a snapshot of ${joined}.`
          : `Created snapshots of ${joined}.`,
      });
    },
    afterFailure: ({ status }: Flux.AfterFailureParams<Base.SnapshotParams>) =>
      addStatus({ ...status, name: rng?.name ?? "" }),
  });
  return ({ selection: { ids }, state: { getResource } }: Tree.ContextMenuProps) => {
    if (rng == null)
      return addStatus({
        variant: "error",
        message: "Failed to snapshot schematics",
        description: "No range is active.",
      });
    const schematics = ids.map((id) => ({
      key: id.key,
      name: getResource(id).name,
    }));
    update({ schematics, range: rng.key });
  };
};

const useRename = Tree.createUseRename({
  useUpdate: Base.useRename,
  resourceID: schematic.resourceID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state,
  } = props;
  const activeRange = Range.useResolve();
  const hasCreatePermission = Access.useCreateGranted(schematic.TYPE_RESOURCE_ID);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const handleDelete = useDelete(props);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const handleCopy = useCopy(props);
  const snapshot = useRangeSnapshot();
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
            onClick={() => group(props)}
          />
        </>
      )}
      <Menu.Divider />
      {resources.every((r) => r.data?.snapshot === false) && hasCreatePermission && (
        <>
          <Range.SnapshotMenuItem range={activeRange} onClick={() => snapshot(props)} />
          <Menu.Item itemKey="copy" onClick={handleCopy}>
            <Icon.Copy />
            Copy
          </Menu.Item>
        </>
      )}
      <Menu.Divider />
      {singleResource && (
        <>
          <Export.ContextMenuItem onClick={() => handleExport(first.id)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, resourceID: firstID })}
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
  type: "schematic",
  icon: <Icon.Schematic />,
  useOnSelect: Panel.useOpenResource,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(resource.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = {
  schematic: TreeItem,
  ...Symbol.TREE_ITEMS,
} satisfies Tree.Items;
