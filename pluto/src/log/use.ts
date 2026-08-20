// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type optional } from "@synnaxlabs/x";
import {
  type Dispatch,
  type SetStateAction,
  useCallback,
  useEffect,
  useMemo,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Channel } from "@/channel";
import { useSyncedRef } from "@/hooks/ref";
import { log } from "@/log/aether";
import { useMemoDeepEqual } from "@/memo";

export interface UseProps
  extends
    optional.Optional<
      Omit<
        z.input<typeof log.logStateZ>,
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
        | "channelDataTypes"
      >,
      "visible"
    >,
    Aether.ComponentProps {
  /** Controlled pause state. When set, the log pauses scrolling while true. */
  hold?: boolean;
  /** Called when the pause state changes from inside the log: a scroll up or the H
   * trigger pauses it, scrolling back to the bottom resumes it. Controlled callers
   * must reflect the value back through hold. */
  onHold?: (hold: boolean) => void;
}

export type LogState = z.output<typeof log.logStateZ>;

export interface UseReturn {
  state: LogState;
  setState: Dispatch<SetStateAction<LogState>>;
}

export const use = ({
  aetherKey,
  font,
  visible = true,
  hideChannelNames = false,
  hideReceiptTimestamp = false,
  timestampPrecision = 0,
  channels: rawChannels,
  color,
  telem,
  hold,
  onHold,
}: UseProps): UseReturn => {
  const channels = rawChannels ?? [];
  const numericChannels = useMemo(
    () =>
      channels
        .map((e) => e.channel)
        .filter((ch): ch is number => typeof ch === "number" && ch > 0),
    [channels],
  );
  const { data: retrievedChannels } = Channel.useResultMultiple({
    keys: numericChannels,
  });
  const channelNames = useMemo(() => {
    const names: Record<string, string> = {};
    if (retrievedChannels != null)
      for (const ch of retrievedChannels) names[String(ch.key)] = ch.name;
    return names;
  }, [retrievedChannels]);
  const channelDataTypes = useMemo(() => {
    const types: Record<string, string> = {};
    if (retrievedChannels != null)
      for (const ch of retrievedChannels) types[String(ch.key)] = String(ch.dataType);
    return types;
  }, [retrievedChannels]);

  const memoProps = useMemoDeepEqual({
    font,
    color,
    telem,
    visible,
    hideChannelNames,
    hideReceiptTimestamp,
    timestampPrecision,
    channelNames,
    channelDataTypes,
    channels,
    ...(hold != null && { scrolling: hold }),
  });

  const holdRef = useSyncedRef(hold);
  // The worker resumes on its own once the user has scrolled back to the bottom. It
  // pushes for other reasons too, so only a value that disagrees with hold is a change.
  const handleAetherChange = useCallback(
    ({ scrolling }: LogState) => {
      const { current: hold } = holdRef;
      if (hold != null && scrolling !== hold) onHold?.(scrolling);
    },
    [onHold],
  );

  const [, state, setState] = Aether.use({
    aetherKey,
    type: log.Log.TYPE,
    schema: log.logStateZ,
    initialState: {
      empty: true,
      region: box.ZERO,
      scrolling: false,
      wheelPos: 0,
      ...memoProps,
    },
    onAetherChange: handleAetherChange,
  });

  useEffect(() => {
    setState((s) => ({ ...s, ...memoProps }));
  }, [memoProps, setState]);

  return { state, setState };
};
