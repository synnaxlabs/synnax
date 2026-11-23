// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import type { FC } from "react";

import { encodes, unbind } from "@/platform/icd/dataType";
import { useMessage } from "@/platform/icd/Formats";
import { jsonFields } from "@/platform/icd/read";
import { suit } from "@/platform/icd/write/write";
import { Task } from "@/platform/task";

export interface SuitProps {
  /** The ICD of the message, which an enclosing WithFormat opens. */
  icdKey: icd.Key;
  path: deep.Path<icd.WriteMessage>;
}

/**
 * Keeps the bindings of a message suited to its format, whose edits can leave bindings
 * that the Core refuses to save. Unbinds each channel that its field no longer takes.
 * It corrects only on a format change, so a refused correction waits for the next one.
 */
export const Suit: FC<SuitProps> = ({ icdKey, path }) => {
  const client = PSynnax.use();
  const { get, set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const handleError = Status.useAsyncErrorHandler();
  const message = useMessage(icdKey, PForm.useFieldValue(path.message));
  useAsyncEffect(
    async (signal) => {
      if (isPreview || client == null || message?.payload.format !== "json") return;
      const fields = jsonFields(message);
      await handleError(async () => {
        const bound = get(path.bindings).value.flatMap((b) =>
          b.type === "channel" ? [b.channel] : [],
        );
        const channels = await client.channels.retrieve(bound);
        if (signal.aborted) return;
        const current = get(path.bindings).value;
        const suited = unbind(suit(current, fields), fields, channels, encodes);
        if (!deep.equal(suited, current)) set(path.bindings, suited);
      }, "Failed to retrieve the channels of a message");
    },
    [isPreview, client, message],
  );
  return null;
};
