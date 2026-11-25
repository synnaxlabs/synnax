// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, Synnax } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { binary, errors } from "@synnaxlabs/x";
import { useState } from "react";

import { type CommandChannelSpec, planCommandChannels } from "@/platform/task/channels";

/**
 * Plans the channels that deploying gives each source of a task, together, so a form
 * can show them first.
 *
 * @param sources - Everything that plan reads from the form for each source, compared
 * by value. Each one must tell its source apart from the others. A change makes the
 * plan again.
 * @param plan - Plans every source, in the order of sources. It should stop between
 * sources once signal aborts, which happens when the sources change.
 * @returns the plan of each source. While a new plan is in flight, a source whose input
 * is unchanged keeps its last plan, and any other has none. With no client, or after a
 * plan fails, no source has one.
 */
export const useChannelPlans = <P>(
  sources: unknown[],
  plan: (client: Synnax, signal: AbortSignal) => Promise<P[]>,
): (P | undefined)[] => {
  const client = PSynnax.use();
  const keys = sources.map((source) => binary.JSON_CODEC.encodeString(source));
  const [plans, setPlans] = useState<ReadonlyMap<string, P>>(() => new Map());
  useAsyncEffect(
    async (signal) => {
      if (client == null) {
        setPlans(new Map());
        return;
      }
      try {
        const next = await plan(client, signal);
        if (!signal.aborted) setPlans(new Map(keys.map((key, i) => [key, next[i]])));
      } catch (e) {
        if (signal.aborted) return;
        setPlans(new Map());
        throw errors.fromUnknown(e);
      }
    },
    [client, JSON.stringify(keys)],
  );
  return keys.map((key) => plans.get(key));
};

const NO_NAMES: ReadonlyMap<string, string> = new Map();

/**
 * Plans the command channels of the write targets of a task, as deploying gives them,
 * so a form can show them first.
 *
 * @param write - The write properties of the device, or undefined before it loads.
 * @param specs - The command channel of each enabled target, in deploy order.
 * @returns the name of each channel that deploying creates, by the properties key of
 * its target, once the plan is done.
 */
export const useCommandChannelNames = (
  write: Record<string, channel.Key> | undefined,
  specs: CommandChannelSpec[],
): ReadonlyMap<string, string> => {
  const [names] = useChannelPlans([{ write, specs }], async (client) =>
    write == null ? [] : [await planCommandChannels(client, write, specs)],
  );
  return names ?? NO_NAMES;
};
