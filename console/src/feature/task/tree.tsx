// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, task } from "@synnaxlabs/client";
import { Access, Icon, Menu, Mosaic, Task as Base } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { useRangeSnapshot } from "@/feature/task/useRangeSnapshot";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Task as PlatformTask } from "@/platform/task";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const openTab = Panel.useOpenTab();
  return useCallback(
    (resource) => openTab({ variant: "resource", resource: resource.id }),
    [openTab],
  );
};

const useDelete = Tree.createUseDelete({
  type: "Task",
  query: Base.useDelete,
  convertKey: String,
});

export const useRename = Tree.createUseRename({
  query: Base.useRename,
  ontologyID: task.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection,
    openTab,
    state: { getResource, shape },
  } = props;
  const { ids, rootID } = selection;
  const resources = getResource(ids);
  const handleDelete = useDelete(props);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = PlatformTask.useExport();
  const snap = useRangeSnapshot();
  const range = Session.Range.useSelectState();
  const group = Group.useCreateFromSelection();
  const rename = useRename(props);
  const ontologyIDs = useMemo(() => ids.map((id) => task.ontologyID(id.key)), [ids]);
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  const hasDeletePermission = Access.useDeleteGranted(ontologyIDs);
  const hasUpdatePermission = Access.useUpdateGranted(ontologyIDs);
  const handleEdit = () => openTab({ variant: "resource", resource: resources[0].id });
  const singleResource = ids.length === 1;
  const hasNoSnapshots = resources.every((r) => r.data?.snapshot === false);
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            onClick={() => group(props)}
          />
          {singleResource && (
            <>
              <ContextMenu.RenameItem onClick={rename} />
              <Menu.Divider />
            </>
          )}
        </>
      )}
      {hasCreatePermission && hasNoSnapshots && range?.persisted === true && (
        <>
          <Range.SnapshotMenuItem
            key="snapshot"
            range={range}
            onClick={() =>
              snap({
                tasks: resources.map(({ id: { key }, name }) => ({ key, name })),
              })
            }
          />
          <Menu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <Menu.Item itemKey="edit" onClick={handleEdit}>
            <Icon.Edit />
            {`${resources[0].data?.snapshot ? "View" : "Edit"} configuration`}
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({ name: resources[0].name, ontologyID: resources[0].id })
            }
          />
          <Export.ContextMenuItem onClick={() => handleExport(ids[0].key)} />
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && (
        <>
          <ContextMenu.DeleteItem onClick={handleDelete} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const TreeItem = Tree.createItem({
  type: "task",
  icon: <Icon.Task />,
  hasChildren: false,
  useOnSelect,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { task: TreeItem } satisfies Tree.Items;
