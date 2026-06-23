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
import { type ReactElement, useCallback } from "react";

import { streamMultiChannelLog } from "@/log/aether/telem/sources";
import { Base, type BaseProps } from "@/log/Base";
import { useRedo, useRetrieveSuspended, useUndo } from "@/log/queries";
import { Triggers } from "@/triggers";

const DEFAULT_RETENTION = TimeSpan.days(1);
const PRELOAD = TimeSpan.seconds(30);

type UndoRedoMode = "undo" | "redo" | "default";

const UNDO_REDO_CONFIG: Triggers.ModeConfig<UndoRedoMode> = {
  undo: [Triggers.UNDO],
  redo: [Triggers.REDO],
  default: [],
  defaultMode: "default",
};

const UNDO_REDO_TRIGGERS = Triggers.flattenConfig(UNDO_REDO_CONFIG);

const useUndoRedoTriggers = (
  key: log.Key,
  enabled?: boolean | (() => boolean),
): void => {
  const { undo } = useUndo({ key });
  const { redo } = useRedo({ key });
  Triggers.use({
    triggers: UNDO_REDO_TRIGGERS,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        if (enabled === false) return;
        if (typeof enabled === "function" && !enabled()) return;
        const mode = Triggers.determineMode(UNDO_REDO_CONFIG, triggers);
        if (mode === "undo") undo();
        else if (mode === "redo") redo();
      },
      [undo, redo, enabled],
    ),
  });
};

export interface LogProps extends Omit<
  BaseProps,
  | "channels"
  | "telem"
  | "hideChannelNames"
  | "hideReceiptTimestamp"
  | "timestampPrecision"
> {
  resourceKey: log.Key;
}

// Log is the connected log visualization. It reads the full log document from the
// Pluto flux store keyed by resourceKey, builds the streaming telemetry source
// internally, and renders the Base primitive. Cmd+Z / Cmd+Shift+Z are wired to undo
// and redo, gated by enableTriggers. The component suspends until the record is in
// cache, so callers must ensure it is loadable (e.g. created on the server).
export const Log = ({
  resourceKey: key,
  enableTriggers,
  ...rest
}: LogProps): ReactElement | null => {
  const { channels, hideChannelNames, hideReceiptTimestamp, timestampPrecision } =
    useRetrieveSuspended({ key });
  useUndoRedoTriggers(key, enableTriggers);
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
      {...rest}
    />
  );
};
