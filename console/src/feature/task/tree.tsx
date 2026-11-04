// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, resource, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Mosaic, Task as Base } from "@synnaxlabs/pluto";

import { useRangeSnapshot } from "@/feature/task/useRangeSnapshot";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Export } from "@/platform/export";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Tree } from "@/platform/tree";

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const openTab = Panel.useOpenTab();
  return (resource) => openTab({ variant: "resource", resource: resource.id });
};

const useDelete = Tree.createUseDelete({
  type: "Task",
  useUpdate: Base.useDelete,
  convertKey: String,
});

const useRename = Tree.createUseRename({
  useUpdate: Base.useRename,
  resourceID: task.resourceID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const { selection, openTab, state } = props;
  const { ids } = selection;
  const resources = Tree.useResources(state, ids);
  const handleDelete = useDelete(props);
  const handleLink = Core.useCopyLinkToClipboard();
  const handleExport = Export.useResource();
  const snap = useRangeSnapshot();
  const range = Range.useResolve();
  const rename = useRename(props);
  const resourceIDs = ids.map((id) => task.resourceID(id.key));
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_RESOURCE_ID);
  const hasDeletePermission = Access.useDeleteGranted(resourceIDs);
  const hasUpdatePermission = Access.useUpdateGranted(resourceIDs);
  const handleEdit = () => openTab({ variant: "resource", resource: resources[0].id });
  const singleResource = ids.length === 1;
  const hasNoSnapshots = resources.every((r) => r.data?.snapshot === false);
  return (
    <ContextMenu.Menu>
      {singleResource && (
        <Menu.Item itemKey="edit" onClick={handleEdit}>
          <Icon.Edit />
          {`${resources[0].data?.snapshot ? "View" : "Edit"} configuration`}
        </Menu.Item>
      )}
      <Menu.Divider />
      {hasUpdatePermission && singleResource && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
      <Menu.Divider />
      {hasCreatePermission && hasNoSnapshots && range?.variant === "persisted" && (
        <Range.SnapshotMenuItem
          key="snapshot"
          range={range}
          onClick={() =>
            snap({
              tasks: resources.map(({ id: { key }, name }) => ({ key, name })),
            })
          }
        />
      )}
      <Menu.Divider />
      {singleResource && (
        <>
          <Export.ContextMenuItem onClick={() => handleExport(ids[0])} />
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({ name: resources[0].name, resourceID: resources[0].id })
            }
          />
        </>
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const TreeItem = Tree.createItem({
  type: "task",
  icon: <Icon.Task />,
  useOnSelect,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(resource.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { task: TreeItem } satisfies Tree.Items;
