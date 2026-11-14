// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, ranger, UnexpectedError } from "@synnaxlabs/client";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Store } from "@synnaxlabs/lyra/store";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Ranger } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CreateChildRangeIcon } from "@/feature/range/ContextMenu";
import { useOpen, useStart, useStop } from "@/feature/range/next/Item";
import { ContextMenu as Base } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Link } from "@/platform/link";
import { Modals } from "@/platform/modals";
import { Range } from "@/platform/range";
import { Session } from "@/session";

/** The menu for ranges in the explorer and the overview. */
export const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const { getItem, subscribe } = List.useUtilContext<ranger.Key, ranger.Range>();
  if (getItem == null || subscribe == null)
    throw new UnexpectedError("No item getter found");
  const ranges = Store.useItems({ keys, subscribe, getItem });
  const single = ranges.length === 1 ? ranges[0] : null;
  const ids = ranger.resourceID(keys);
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const canPlot = Access.useCreateGranted(lineplot.TYPE_RESOURCE_ID);
  const favoriteKeys = Session.Range.useSelectKeys();
  const dispatch = Session.useDispatch();
  const open = useOpen();
  const plot = Range.useAddToNewPlot();
  const start = useStart();
  const stop = useStop();
  const openCreate = Range.useCreateModal();
  const copyLink = Core.useCopyLinkToClipboard();
  const handleError = Status.useErrorHandler();
  const confirm = Modals.useConfirmDelete({
    type: "Range",
    description: "Deleting a range also deletes its child ranges.",
  });
  const { update: del } = Ranger.useDelete();
  const now = useNow(() => null);
  const inStage = (stage: ranger.Stage) =>
    ranges.filter((r) => Ranger.getStage(r.timeRange, now) === stage);
  const planned = inStage("to_do");
  const running = inStage("in_progress");
  const handleDelete = () =>
    handleError(async () => {
      if (!(await confirm(ranges))) return;
      const keys = ranges.map((r) => r.key);
      dispatch(Session.Range.remove({ keys }));
      del(keys);
    }, "Failed to delete range");
  return (
    <Base.Menu>
      {single != null && (
        <Menu.Item itemKey="open" onClick={() => open(single.key)}>
          <Icon.Details />
          Open
        </Menu.Item>
      )}
      {canPlot && ranges.length > 0 && (
        <Menu.Item itemKey="plot" onClick={() => plot(keys)}>
          <Icon.LinePlot />
          Plot
        </Menu.Item>
      )}
      <Menu.Divider />
      {planned.length > 0 && canUpdate && (
        <Menu.Item itemKey="start" onClick={() => planned.forEach(start)}>
          <Icon.Play />
          Start
        </Menu.Item>
      )}
      {running.length > 0 && canUpdate && (
        <Menu.Item itemKey="stop" onClick={() => running.forEach(stop)}>
          <Icon.Stop />
          Stop
        </Menu.Item>
      )}
      {single != null && canCreate && (
        <Menu.Item
          itemKey="startChild"
          onClick={() => openCreate({ parent: single.key })}
        >
          <CreateChildRangeIcon />
          Start child range
        </Menu.Item>
      )}
      {single != null && canUpdate && (
        <Base.RenameItem onClick={() => Text.edit(List.itemNameID(single.key))} />
      )}
      <Menu.Divider />
      {ranges.some((r) => !favoriteKeys.includes(r.key)) && (
        <Menu.Item
          itemKey="favorite"
          onClick={() =>
            Session.Range.fromClient(ranges).forEach((r) =>
              dispatch(Session.Range.add(r)),
            )
          }
        >
          <Icon.StarFilled />
          Pin
        </Menu.Item>
      )}
      {ranges.some((r) => favoriteKeys.includes(r.key)) && (
        <Menu.Item
          itemKey="unfavorite"
          onClick={() =>
            dispatch(Session.Range.remove({ keys: ranges.map((r) => r.key) }))
          }
        >
          <Icon.StarOutlined />
          Unpin
        </Menu.Item>
      )}
      {single != null && (
        <Link.CopyContextMenuItem
          onClick={() =>
            copyLink({ name: single.name, resourceID: ranger.resourceID(single.key) })
          }
        />
      )}
      <Menu.Divider />
      {canDelete && ranges.length > 0 && <Base.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <Base.ReloadConsoleItem />
    </Base.Menu>
  );
};
