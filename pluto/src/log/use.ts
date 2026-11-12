// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log as clientLog } from "@synnaxlabs/client";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { box, type optional, TimeSpan } from "@synnaxlabs/x";
import { type Dispatch, type SetStateAction, useEffect, useRef } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Channel } from "@/channel";
import { log } from "@/log/aether";

// Formatting the entry cap takes seconds on a slow machine.
const COPY_TIMEOUT = TimeSpan.seconds(30);

export interface UseProps
  extends
    optional.Optional<
      Omit<
        z.input<typeof log.logStateZ>,
        | "channels"
        | "region"
        | "scrollPosition"
        | "scrollback"
        | "empty"
        | "scrolling"
        | "resumedAt"
        | "wheelPos"
        | "selectionStart"
        | "selectionEnd"
        | "visibleStart"
        | "rowEntries"
        | "computedLineHeight"
        | "channelNames"
        | "channelDataTypes"
        | "position"
      >,
      "visible"
    >,
    Aether.ComponentProps {
  channels?: clientLog.ChannelEntry[];
  /** Controlled pause state. When set, the log pauses scrolling while true. */
  hold?: boolean;
  /** Called when the pause state changes from inside the log: a scroll up or the H
   * trigger pauses it, scrolling back to the bottom resumes it. Controlled callers
   * must reflect the value back through hold. */
  onHold?: (hold: boolean) => void;
  /** Where a log that mounts paused opens, as reported to onLeave. */
  position?: string;
  /** Called when the log unmounts, with the position of a paused view. A live log
   * reports an empty position. */
  onLeave?: (position: string) => void;
}

export type LogState = z.output<typeof log.logStateZ>;

export interface UseReturn {
  state: LogState;
  setState: Dispatch<SetStateAction<LogState>>;
  methods: { selection: () => Promise<log.Selection> };
}

export const use = ({
  aetherKey,
  font,
  visible = true,
  channelNamesHidden = false,
  timestampsHidden = false,
  timestampPrecision = 0,
  timeZone = "local",
  channels: rawChannels,
  color,
  telem,
  hold,
  onHold,
  position,
  onLeave,
}: UseProps): UseReturn => {
  const channels = rawChannels ?? [];
  const { data: retrievedChannels } = Channel.useResultMultiple({
    where: { key: { in: channels.map((e) => e.channel) } },
  });
  const names: Record<string, string> = {};
  if (retrievedChannels != null)
    for (const ch of retrievedChannels) names[String(ch.key)] = ch.name;
  const channelNames = names;
  const types: Record<string, string> = {};
  if (retrievedChannels != null)
    for (const ch of retrievedChannels) types[String(ch.key)] = String(ch.dataType);
  const channelDataTypes = types;

  const memoProps = useMemoDeepEqual({
    font,
    color,
    telem,
    visible,
    channelNamesHidden,
    timestampsHidden,
    timestampPrecision,
    timeZone,
    channelNames,
    channelDataTypes,
    channels,
    ...(hold != null && { scrolling: hold }),
  });

  const holdRef = useSyncedRef(hold);
  const resumedAtRef = useRef(0);
  // The worker stamps resumedAt when it resumes on its own. Its pushes can still echo
  // a stale scrolling value after a hold change, so the stamp is the only signal.
  const handleAetherChange = ({ resumedAt }: LogState) => {
    if (resumedAt <= resumedAtRef.current) return;
    resumedAtRef.current = resumedAt;
    if (holdRef.current != null) onHold?.(false);
  };

  const [, state, setState, methods] = Aether.use({
    aetherKey,
    type: log.Log.TYPE,
    schema: log.logStateZ,
    methods: log.methodsZ,
    invokeTimeout: COPY_TIMEOUT,
    initialState: {
      empty: true,
      region: box.ZERO,
      scrolling: false,
      wheelPos: 0,
      position,
      ...memoProps,
    },
    onAetherChange: handleAetherChange,
  });

  useEffect(() => {
    setState((s) => ({ ...s, ...memoProps }));
  }, [memoProps, setState]);

  const stateRef = useSyncedRef(state);
  const onLeaveRef = useSyncedRef(onLeave);
  useEffect(
    () => () => onLeaveRef.current?.(stateRef.current.position),
    [onLeaveRef, stateRef],
  );

  return { state, setState, methods };
};
