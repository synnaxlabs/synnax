// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, ranger, UnexpectedError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Store } from "@synnaxlabs/lyra/store";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Ranger } from "@synnaxlabs/pluto";
import { strings, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CreateChildRangeIcon } from "@/feature/range/ContextMenu";
import { useOpen, useStop } from "@/feature/range/next/Item";
import { ContextMenu as Base } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { LinePlot } from "@/platform/lineplot";
import { Link } from "@/platform/link";
import { Modals } from "@/platform/modals";
import { Range } from "@/platform/range";
import { Session } from "@/session";

/** Creates a line plot of the ranges, and opens it. */
export const usePlot = (): ((ranges: ranger.Range[]) => void) => {
  const create = LinePlot.useCreate();
  return (ranges) =>
    create({
      name: strings.naturalLanguageJoin(
        ranges.map((r) => r.name),
        "range",
      ),
      ranges: ranges.map(({ key }) => ({ variant: "key", key })),
    });
};

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
  const plot = usePlot();
  const stop = useStop();
  const openCreate = Range.useCreateModal();
  const copyLink = Core.useCopyLinkToClipboard();
  const handleError = Status.useErrorHandler();
  const confirm = Modals.useConfirmDelete({
    type: "Range",
    description: "Deleting a range also deletes its child ranges.",
  });
  const { update: del } = Ranger.useDelete();
  const now = TimeStamp.now();
  const running = ranges.filter(
    (r) => Ranger.getStage(r.timeRange, now) === "in_progress",
  );
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
        <Menu.Item itemKey="plot" onClick={() => plot(ranges)}>
          <Icon.LinePlot />
          Plot
        </Menu.Item>
      )}
      <Menu.Divider />
      {running.length > 0 && canUpdate && (
        <Menu.Item itemKey="stop" onClick={() => running.forEach(stop)}>
          <Icon.Stop />
          Stop
        </Menu.Item>
      )}
      {single != null && canCreate && (
        <Menu.Item
          itemKey="createChild"
          onClick={() => openCreate({ parent: single.key })}
        >
          <CreateChildRangeIcon />
          Create child range
        </Menu.Item>
      )}
      {single != null && canUpdate && (
        <Base.RenameItem onClick={() => Text.edit(List.itemNameID(single.key))} />
      )}
      <Menu.Divider />
      <Base.FavoriteItems
        anyFavorited={ranges.some((r) => favoriteKeys.includes(r.key))}
        anyNotFavorited={ranges.some((r) => !favoriteKeys.includes(r.key))}
        onFavorite={() =>
          Session.Range.fromClient(ranges).forEach((r) =>
            dispatch(Session.Range.add(r)),
          )
        }
        onUnfavorite={() =>
          dispatch(Session.Range.remove({ keys: ranges.map((r) => r.key) }))
        }
      />
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
