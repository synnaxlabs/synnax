// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, icd } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useRef } from "react";

import { jsonFields } from "@/platform/icd/read";
import { Task } from "@/platform/task";

export interface UseSuitParams<V> {
  /** The form path of the value to keep suited. */
  path: deep.Target<V>;
  /** The message whose format the value must suit. */
  message: icd.Message | null;
  /** @returns value, changed to suit fields. */
  suit: (value: V, fields: icd.JSONField[]) => V;
  /** @returns the channels that value binds. */
  bound: (value: V) => channel.Key[];
  /** @returns value, with each channel that its field does not take unbound. */
  unbind: (value: V, fields: icd.JSONField[], channels: channel.Channel[]) => V;
}

interface Sent<V> {
  message: icd.Message;
  value: V;
}

/**
 * Keeps the value at path suited to a JSON message, whose edits can leave a value that
 * the Core refuses to save. It suits the value at once, then unbinds once the bound
 * channels are retrieved. A failed retrieval skips only the unbinding. A correction
 * that the Core refuses waits for the next change of the message.
 */
export const useSuit = <V>({
  path,
  message,
  suit,
  bound,
  unbind,
}: UseSuitParams<V>) => {
  const client = PSynnax.use();
  const { get, set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const handleError = Status.useAsyncErrorHandler();
  const value = PForm.useFieldValue(path);
  const sent = useRef<Sent<V> | null>(null);
  useAsyncEffect(
    async (signal) => {
      if (isPreview || message?.payload.format !== "json") return;
      const last = sent.current;
      // The form changes the value away from a correction only when it undoes a
      // refused save, or when the user edits the value.
      if (last?.message === message && !deep.equal(last.value, get(path).value)) return;
      const fields = jsonFields(message);
      const correct = (next: V) => {
        if (deep.equal(next, get(path).value)) return;
        sent.current = { message, value: next };
        set(path, next);
      };
      correct(suit(get(path).value, fields));
      if (client == null) return;
      await handleError(async () => {
        const channels = await client.channels.retrieve(bound(get(path).value));
        if (!signal.aborted) correct(unbind(get(path).value, fields, channels));
      }, "Failed to retrieve the channels of a message");
    },
    [isPreview, client, message, value],
  );
};
