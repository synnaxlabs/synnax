// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, query, task } from "@synnaxlabs/client";
import { Access, Arc, Icon, Menu, Task } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";

import { Core } from "@/platform/core";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useRename = Tree.createUseRename({
  query: Arc.useRename,
  ontologyID: arc.ontologyID,
  convertKey: String,
});

const useDelete = Tree.createUseDelete({
  type: "Arc",
  query: Arc.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, store }) => {
    store.dispatch(Session.Arc.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    client,
    selection: { ids },
    state: { getResource },
  } = props;
  const keys = ids.map((id) => id.key);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const canControl = Framer.useCanCommand();
  const { update: runCommand } = Task.useCommand();
  const rename = useRename(props);
  const handleDelete = useDelete(props);
  const handleLink = Core.useCopyLinkToClipboard();
  const firstID = ids[0];
  const first = getResource(firstID);
  const singleResource = ids.length === 1;
  const redeployTaskKeys = !canControl
    ? []
    : keys
        .map((key) => client.arcs.task.getCached(key))
        .filter(
          (tsk): tsk is task.Task =>
            query.isLive(tsk) && tsk != null && task.drifted(tsk),
        )
        .map(({ key }) => key);
  return (
    <ContextMenu.Menu>
      {redeployTaskKeys.length > 0 && (
        <Menu.Item
          className={CSS.BE("arc", "redeploy-item")}
          itemKey="redeploy"
          onClick={() =>
            runCommand(redeployTaskKeys.map((key) => ({ task: key, type: "start" })))
          }
        >
          <Icon.Refresh />
          Redeploy
        </Menu.Item>
      )}
      <Menu.Divider />
      {hasUpdatePermission && singleResource && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
      <Menu.Divider />
      {singleResource && (
        <Link.CopyContextMenuItem
          onClick={() => handleLink({ name: first.name, ontologyID: firstID })}
        />
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const TreeItem = Tree.createItem({
  type: "arc",
  icon: <Icon.Arc />,
  canDrop: () => true,
  useOnSelect: Panel.useOpenResource,
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { arc: TreeItem } satisfies Tree.Items;
