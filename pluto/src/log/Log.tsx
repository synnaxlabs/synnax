// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log, type ranger } from "@synnaxlabs/client";
import { type Menu } from "@synnaxlabs/lyra/menu";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";

import { streamMultiChannelLog } from "@/log/aether/telem/sources";
import { Base, type BaseProps } from "@/log/Base";
import { use, useRedo, useUndo } from "@/log/queries";
import { useKey } from "@/log/Suspended";
import { use as useRange } from "@/ranger/queries";

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
  | "range"
> {}

interface RangeProps extends BaseProps {
  rangeKey: ranger.Key;
}

const Range = ({ rangeKey, ...rest }: RangeProps): ReactElement => {
  const { timeRange } = useRange({ key: rangeKey });
  return <Base range={timeRange} {...rest} />;
};

// Log is the connected log visualization. It reads the full log document from the
// Pluto flux store keyed by the surrounding Log scope, builds the streaming telemetry
// source internally, and renders the Base primitive. Cmd+Z / Cmd+Shift+Z are wired to
// undo and redo, gated by enableTriggers. A log that a range owns passes that range to
// Base. The component suspends until the record is in cache, so callers must render it
// within a Log.Suspended boundary.
export const Log = ({ enableTriggers, ...rest }: LogProps): ReactElement | null => {
  const key = useKey();
  const {
    channels,
    channelNamesHidden,
    timestampsHidden,
    timestampPrecision,
    timeZone,
    range,
  } = use({ key });
  const undoRedo = useUndoRedo(key, enableTriggers);
  const telem = streamMultiChannelLog({ channels: [] });
  const props: BaseProps = {
    telem,
    channels,
    channelNamesHidden,
    timestampsHidden,
    timestampPrecision,
    timeZone,
    enableTriggers,
    undoRedo,
    ...rest,
  };
  if (range == null) return <Base {...props} />;
  return <Range rangeKey={range} {...props} />;
};
