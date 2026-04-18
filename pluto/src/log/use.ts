// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type optional } from "@synnaxlabs/x";
import { type Dispatch, type SetStateAction, useEffect, useMemo } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Channel } from "@/channel";
import { log } from "@/log/aether";
import { useMemoDeepEqual } from "@/memo";

export interface UseProps
  extends
    optional.Optional<
      Omit<
        z.input<typeof log.logState>,
        | "region"
        | "scrollPosition"
        | "scrollback"
        | "empty"
        | "scrolling"
        | "wheelPos"
        | "selectionStart"
        | "selectionEnd"
        | "visibleStart"
        | "selectedText"
        | "selectedLines"
        | "computedLineHeight"
        | "channelNames"
      >,
      "visible"
    >,
    Aether.ComponentProps {}

export type LogState = z.output<typeof log.logState>;

export interface UseReturn {
  state: LogState;
  setState: Dispatch<SetStateAction<LogState>>;
}

export const use = ({
  aetherKey,
  font,
  visible = true,
  showChannelNames = true,
  showReceiptTimestamp = true,
  timestampPrecision = 0,
  channels = [],
  color,
  telem,
}: UseProps): UseReturn => {
  const numericChannels = useMemo(
    () =>
      channels
        .map((e) => e.channel)
        .filter((ch): ch is number => typeof ch === "number" && ch > 0),
    [channels],
  );
  const { data: retrievedChannels } = Channel.useRetrieveMultiple({
    keys: numericChannels,
  });
  const channelNames = useMemo(() => {
    const names: Record<string, string> = {};
    if (retrievedChannels != null)
      for (const ch of retrievedChannels) names[String(ch.key)] = ch.name;
    return names;
  }, [retrievedChannels]);

  const memoProps = useMemoDeepEqual({
    font,
    color,
    telem,
    visible,
    showChannelNames,
    showReceiptTimestamp,
    timestampPrecision,
    channelNames,
    channels,
  });

  const [, state, setState] = Aether.use({
    aetherKey,
    type: log.Log.TYPE,
    schema: log.logState,
    initialState: {
      empty: true,
      region: box.ZERO,
      scrolling: false,
      wheelPos: 0,
      ...memoProps,
    },
  });

  useEffect(() => {
    setState((s) => ({ ...s, ...memoProps }));
  }, [memoProps, setState]);

  return { state, setState };
};
