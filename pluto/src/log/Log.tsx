// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { type Menu } from "@synnaxlabs/lyra/menu";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";

import { canDropHaulItem, filterHaulItems } from "@/channel/types";
import { CSS } from "@/css";
import { streamMultiChannelLog } from "@/log/aether/telem/sources";
import { Base, type BaseProps } from "@/log/Base";
import { use, useRedo, useSingleDispatch, useUndo } from "@/log/queries";
import { useKey } from "@/log/Suspended";

const useUndoRedo = (
  key: log.Key,
  enabled?: Triggers.Condition,
): Menu.UndoRedoItemsProps => {
  const { undo, canUndo } = useUndo({ key });
  const { redo, canRedo } = useRedo({ key });
  Triggers.useUndoRedo({ undo, redo, enabled });
  return { undo, redo, canUndo, canRedo };
};

export interface LogProps extends Omit<
  BaseProps,
  | "channels"
  | "telem"
  | "channelNamesHidden"
  | "timestampsHidden"
  | "timestampPrecision"
  | "timeZone"
> {
  /** Whether dropped channels join the log. */
  editable?: boolean;
}

// Log is the connected log visualization. It reads the full log document from the
// Pluto flux store keyed by the surrounding Log scope, builds the streaming telemetry
// source internally, and renders the Base primitive. Cmd+Z / Cmd+Shift+Z are wired to
// undo and redo, gated by enableTriggers. The component suspends until the record is in
// cache, so callers must render it within a Log.Suspended boundary.
export const Log = ({
  enableTriggers,
  editable = true,
  className,
  ...rest
}: LogProps): ReactElement | null => {
  const key = useKey();
  const {
    channels,
    channelNamesHidden,
    timestampsHidden,
    timestampPrecision,
    timeZone,
  } = use({ key });
  const undoRedo = useUndoRedo(key, enableTriggers);
  const telem = streamMultiChannelLog({ channels: [] });
  const dispatch = useSingleDispatch();
  const canDrop: Haul.CanDrop = (state) => editable && canDropHaulItem(state);
  const dropProps = Haul.useDrop({
    type: "channel_log",
    canDrop,
    onDrop: ({ items }) => {
      const dropped = filterHaulItems(items);
      const present = new Set(channels.map((e) => e.channel));
      const additions = dropped.filter(({ key }) => !present.has(key));
      if (additions.length > 0)
        dispatch(additions.map(({ key }) => log.addChannel({ channel: key })));
      return dropped;
    },
  });
  const dragging = Haul.useDraggingState();
  return (
    <Base
      {...dropProps}
      className={CSS.cls(CSS.dropRegion(canDrop(dragging)), className)}
      telem={telem}
      channels={channels}
      channelNamesHidden={channelNamesHidden}
      timestampsHidden={timestampsHidden}
      timestampPrecision={timestampPrecision}
      timeZone={timeZone}
      enableTriggers={enableTriggers}
      undoRedo={undoRedo}
      {...rest}
    />
  );
};
