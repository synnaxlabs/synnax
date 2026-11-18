// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/range/menu/ContextMenu.css";

import { lineplot, ranger } from "@synnaxlabs/client";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Ranger } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode } from "react";

import { ContextMenu as Base } from "@/platform/context-menu";
import { Framer } from "@/platform/framer";
import { Modals } from "@/platform/modals";
import { AddToPlot } from "@/platform/range/menu/AddToPlot";
import { Color } from "@/platform/range/menu/Color";
import { Copy } from "@/platform/range/menu/Copy";
import { Labels } from "@/platform/range/menu/Labels";
import { Parent } from "@/platform/range/menu/Parent";
import { NextIcon, useStart, useStop } from "@/platform/range/stage";
import { useAddToNewPlot } from "@/platform/range/useAddToNewPlot";
import { useCreateModal } from "@/platform/range/useCreateModal";
import { useOpen } from "@/platform/range/useOpen";
import { Session } from "@/session";

export interface ContextMenuProps {
  /** The keys of the ranges the menu acts on. */
  keys: ranger.Key[];
  /** Items the surface puts first, such as the action a click on the range does. */
  primaryMenuItems?: ReactNode;
  /** Edits the name of the range in place. Without it, Rename is absent. */
  onRename?: (range: ranger.Range) => void;
  /** Starts a child of the range in place. Without it, the create form opens. */
  onStartChild?: (range: ranger.Range) => void;
  /** Stops a running range and starts the one after it. Without it, Next is absent. */
  onNext?: (range: ranger.Range) => void;
  /** Downloads the data of the ranges. Without it, the CSV download form opens. */
  onDownloadCSV?: (ranges: ranger.Range[]) => void;
  /** Items the surface adds among the exports, such as an upload. */
  extraMenuItems?: ReactNode;
}

/** The menu for ranges, wherever a range shows: lists, the overview, and plots. */
export const ContextMenu = ({
  keys,
  primaryMenuItems,
  onRename,
  onStartChild,
  onNext,
  onDownloadCSV,
  extraMenuItems,
}: ContextMenuProps): ReactElement => {
  const ranges = Ranger.useResultMultiple({ keys }).data ?? [];
  const single = ranges.length === 1 ? ranges[0] : null;
  const ids = ranger.resourceID(keys);
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const canPlot = Access.useCreateGranted(lineplot.TYPE_RESOURCE_ID);
  const favoriteKeys = Session.Range.useSelectFavoriteKeys();
  const dispatch = Session.useDispatch();
  const open = useOpen();
  const plot = useAddToNewPlot();
  const start = useStart();
  const stop = useStop();
  const openCreate = useCreateModal();
  const openCSV = Framer.useDownloadCSVModal();
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
      dispatch(Session.Range.remove({ keys }));
      del(keys);
    }, "Failed to delete range");
  const handleCSV = () => {
    if (onDownloadCSV != null) return onDownloadCSV(ranges);
    if (single == null) return;
    openCSV({
      timeRange: single.timeRange.numeric,
      name: single.name,
      channels: [],
      icon: <Icon.Range />,
    });
  };
  // A click on empty space, or ranges that still load, leave nothing to act on.
  if (ranges.length === 0)
    return (
      <Base.Menu>
        <Base.ReloadConsoleItem />
      </Base.Menu>
    );
  return (
    <Base.Menu>
      {primaryMenuItems}
      {single != null && (
        <Menu.Item itemKey="open" onClick={() => open(single.key)}>
          <Icon.Details />
          Open
        </Menu.Item>
      )}
      {canPlot && (
        <>
          <Menu.Item itemKey="plot" onClick={() => plot(keys)}>
            <Icon.LinePlot />
            Plot
          </Menu.Item>
          <AddToPlot ranges={ranges} />
        </>
      )}
      <Menu.Divider />
      {single != null && canUpdate && onRename != null && (
        <Base.RenameItem onClick={() => onRename(single)} />
      )}
      {canUpdate && (
        <>
          <Labels ranges={ranges} />
          <Color ranges={ranges} />
          <Parent ranges={ranges} />
        </>
      )}
      <Base.FavoriteItems
        anyFavorited={ranges.some((r) => favoriteKeys.includes(r.key))}
        anyNotFavorited={ranges.some((r) => !favoriteKeys.includes(r.key))}
        onFavorite={() =>
          Session.Range.fromClient(ranges).forEach((r) =>
            dispatch(Session.Range.add(r)),
          )
        }
        onUnfavorite={() => dispatch(Session.Range.remove({ keys }))}
      />
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
      {onNext != null &&
        single != null &&
        single.parent != null &&
        running.length === 1 &&
        canUpdate &&
        canCreate && (
          <Menu.Item itemKey="next" onClick={() => onNext(single)}>
            <NextIcon />
            Next
          </Menu.Item>
        )}
      {single != null && canCreate && (
        <Menu.Item
          itemKey="startChild"
          onClick={() =>
            onStartChild == null
              ? void openCreate({ parent: single.key })
              : onStartChild(single)
          }
        >
          <Ranger.CreateIcon />
          Start child range
        </Menu.Item>
      )}
      <Menu.Divider />
      <Copy ranges={ranges} />
      {(single != null || onDownloadCSV != null) && (
        <Menu.Item itemKey="csv" onClick={handleCSV}>
          <Icon.CSV />
          Download as CSV
        </Menu.Item>
      )}
      {extraMenuItems}
      <Menu.Divider />
      {canDelete && <Base.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <Base.ReloadConsoleItem />
    </Base.Menu>
  );
};
