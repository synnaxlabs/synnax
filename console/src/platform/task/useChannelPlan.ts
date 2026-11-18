// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useState } from "react";

/**
 * Plans the channels that deploying gives a task, so a form can show them first.
 *
 * @param inputs - Everything that plan reads from the form, compared by value. A change
 * makes the plan again.
 * @returns the plan, or undefined with no client, before the first plan, and after one
 * fails.
 */
export const useChannelPlan = <P>(
  inputs: unknown,
  plan: (client: Synnax) => Promise<P>,
): P | undefined => {
  const client = PSynnax.use();
  const [result, setResult] = useState<P>();
  useAsyncEffect(
    async (signal) => {
      if (client == null) {
        setResult(undefined);
        return;
      }
      try {
        const next = await plan(client);
        if (!signal.aborted) setResult(next);
      } catch (e) {
        if (!signal.aborted) setResult(undefined);
        throw errors.fromUnknown(e);
      }
    },
    [client, JSON.stringify(inputs)],
  );
  return result;
};
