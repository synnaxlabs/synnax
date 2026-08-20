// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log } from "@synnaxlabs/client";
import { primitive, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { streamMultiChannelLog } from "@/log/aether/telem/sources";
import { Base, type BaseProps } from "@/log/Base";
import { use, useRedo, useUndo } from "@/log/queries";
import { useKey } from "@/log/Suspended";
import { type Menu } from "@/menu";
import { Triggers } from "@/triggers";

const DEFAULT_RETENTION = TimeSpan.days(7);
const PRELOAD = TimeSpan.seconds(30);

const useUndoRedo = (
  key: log.Key,
  enabled?: Triggers.Condition,
): Menu.UndoRedoItemsProps => {
  const { undo, canUndo } = useUndo({ key });
  const { redo, canRedo } = useRedo({ key });
  Triggers.useUndoRedo({ undo, redo, enabled });
  return useMemo(
    () => ({ undo, redo, canUndo, canRedo }),
    [undo, redo, canUndo, canRedo],
  );
};

export interface LogProps extends Omit<
  BaseProps,
  | "channels"
  | "telem"
  | "hideChannelNames"
  | "hideReceiptTimestamp"
  | "timestampPrecision"
> {}

// Log is the connected log visualization. It reads the full log document from the
// Pluto flux store keyed by the surrounding Log scope, builds the streaming telemetry
// source internally, and renders the Base primitive. Cmd+Z / Cmd+Shift+Z are wired to
// undo and redo, gated by enableTriggers. The component suspends until the record is in
// cache, so callers must render it within a Log.Suspended boundary.
export const Log = ({ enableTriggers, ...rest }: LogProps): ReactElement | null => {
  const key = useKey();
  const { channels, hideChannelNames, hideReceiptTimestamp, timestampPrecision } = use({
    key,
  });
  const undoRedo = useUndoRedo(key, enableTriggers);
  // A channel entry with key 0 is an unconfigured placeholder row; the telem source
  // must not subscribe to it.
  const activeChannels = channels.filter((e) => !primitive.isZero(e.channel));
  const telem = streamMultiChannelLog({
    channels: [],
    timeSpan: PRELOAD,
    keepFor: DEFAULT_RETENTION,
  });
  return (
    <Base
      telem={telem}
      channels={activeChannels}
      hideChannelNames={hideChannelNames}
      hideReceiptTimestamp={hideReceiptTimestamp}
      timestampPrecision={timestampPrecision}
      enableTriggers={enableTriggers}
      undoRedo={undoRedo}
      {...rest}
    />
  );
};
