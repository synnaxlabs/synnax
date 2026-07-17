// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax as Client, task } from "@synnaxlabs/client";
import {
  Access,
  Icon,
  Menu,
  Mosaic,
  Status,
  Synnax,
  Task as Base,
} from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { retrieveAndPlaceLayout } from "@/feature/task/layouts";
import { useRangeSnapshot } from "@/feature/task/useRangeSnapshot";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Layout } from "@/platform/layout";
import { Link } from "@/platform/link";
import { Range } from "@/platform/range";
import { Task as PlatformTask } from "@/platform/task";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const openTask = (
  client: Client,
  key: string,
  name: string,
  placeLayout: Layout.Placer,
  handleError: Status.ErrorHandler,
): void =>
  handleError(
    async () => await retrieveAndPlaceLayout(client, key, placeLayout),
    `Could not open ${name}`,
  );

const useOnSelect = (): ((entry: Tree.Entry) => void) => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (entry) => {
      if (client == null) return;
      openTask(client, entry.id.key, entry.name, placeLayout, handleError);
    },
    [client, placeLayout, handleError],
  );
};

const useName: Tree.UseName = (id) =>
  Base.useRetrieve({ key: id.key }).data?.name ?? "";

const useDelete = Tree.createUseDelete({
  type: "Task",
  query: Base.useDelete,
  convertKey: String,
  useName,
  beforeUpdate: async ({ data, removeLayout }) => {
    removeLayout(...data);
    return data;
  },
});

export const useRename = Tree.createUseRename({
  query: Base.useRename,
  ontologyID: task.ontologyID,
  convertKey: String,
  useName,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    const layout = Session.Layout.selectByFilter(
      store.getState(),
      (l) => (l.args as PlatformTask.FormLayoutArgs)?.taskKey === key,
    );
    if (layout != null) {
      store.dispatch(Session.Layout.rename({ key: layout.key, name }));
      rollbacks.push(() => Session.Layout.rename({ key: layout.key, name: oldName }));
    }
    return { ...data, name };
  },
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection,
    client,
    placeLayout,
    handleError,
    state: { shape },
  } = props;
  const { ids, rootID } = selection;
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
  const tasks = Base.useRetrieveMultiple({ keys: ids.map((id) => id.key) }).data;
  const hasNoSnapshots = tasks?.every((t) => t.snapshot === false) ?? false;
  const firstTask = tasks?.find((t) => t.key === ids[0]?.key);
  const handleEdit = () => {
    if (client == null) return;
    openTask(client, ids[0].key, firstTask?.name ?? "", placeLayout, handleError);
  };
  const singleResource = ids.length === 1;
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
                tasks: ids.map((id) => ({
                  key: id.key,
                  name: tasks?.find((t) => t.key === id.key)?.name ?? "",
                })),
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
            {`${firstTask?.snapshot ? "View" : "Edit"} configuration`}
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({ name: firstTask?.name ?? "", ontologyID: ids[0] })
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
  useName,
  useOnSelect,
  haulItems: (id) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { task: TreeItem } satisfies Tree.Items;
