// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/arc/ContextMenu.css";

import { arc, query, task } from "@synnaxlabs/client";
import {
  Access,
  Arc,
  type Flux,
  Icon,
  type List,
  Menu,
  Status,
  Synnax,
  Task,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Core } from "@/platform/core";
import { ContextMenu as Base } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Link } from "@/platform/link";
import { Modals } from "@/platform/modals";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";

export interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: List.GetItem<arc.Key, arc.Arc>;
  textIdPrefix?: string;
}

export const ContextMenu = ({
  keys,
  getItem,
  textIdPrefix = "text",
}: ContextMenuProps) => {
  const ids = arc.ontologyID(keys);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const someSelected = keys.length > 0;
  const isSingle = keys.length === 1;

  const client = Synnax.use();
  const canControl = Framer.useCanCommand();
  const { update: runCommand } = Task.useCommand();
  const redeployTaskKeys =
    client == null || !canControl
      ? []
      : keys
          .map((key) => client.arcs.task.getCached(key))
          .filter(
            (tsk): tsk is task.Task =>
              query.isLive(tsk) && tsk != null && task.drifted(tsk),
          )
          .map(({ key }) => key);

  const dispatch = Session.useDispatch();
  const openTab = Panel.useOpenTab();
  const addStatus = Status.useAdder();
  const handleLink = Core.useCopyLinkToClipboard();
  const confirm = Modals.useConfirmDelete({ type: "Arc" });
  const { update: del } = Arc.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<arc.Key | arc.Key[]>) => {
        const arcKeys = array.toArray(data);
        if (arcKeys.length === 0) return false;
        const arcs = getItem(arcKeys);
        if (!(await confirm(arcs))) return false;
        dispatch(Session.Arc.remove({ keys: arcKeys }));
        return data;
      },
      [getItem, dispatch],
    ),
  });

  const handleEdit = () => {
    const retrieved = getItem(keys[0]);
    if (retrieved == null)
      return addStatus({
        variant: "error",
        message: "Failed to open Arc editor",
        description: "The Arc no longer exists.",
      });
    openTab({ variant: "resource", resource: arc.ontologyID(retrieved.key) });
  };

  const handleCopyLink = () => {
    const name = getItem(keys[0])?.name;
    if (name == null) return;
    handleLink({ name, ontologyID: arc.ontologyID(keys[0]) });
  };

  return (
    <Base.Menu>
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
      {hasUpdatePermission && isSingle && (
        <>
          <Menu.Item itemKey="edit" onClick={handleEdit}>
            <Icon.Edit />
            Edit
          </Menu.Item>
          <Menu.Divider />
          <Base.RenameItem onClick={() => Text.edit(`${textIdPrefix}-${keys[0]}`)} />
        </>
      )}
      <Menu.Divider />
      {isSingle && <Link.CopyContextMenuItem onClick={handleCopyLink} />}
      <Menu.Divider />
      {hasDeletePermission && someSelected && (
        <Base.DeleteItem onClick={() => del(keys)} />
      )}
      <Menu.Divider />
      <Base.ReloadConsoleItem />
    </Base.Menu>
  );
};
