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
import type { deep } from "@synnaxlabs/x";
import type { FC } from "react";

import { encodes, unbind } from "@/platform/icd/dataType";
import { useMessage } from "@/platform/icd/Formats";
import { useSuit } from "@/platform/icd/useSuit";
import { suit } from "@/platform/icd/write/write";

export interface SuitProps {
  /** The ICD of the message, which an enclosing WithFormats opens. */
  icdKey: icd.Key;
  path: deep.Path<icd.WriteMessage>;
}

/**
 * Keeps the bindings of a message suited to its format, and unbinds each channel that
 * its field no longer takes.
 */
export const Suit: FC<SuitProps> = ({ icdKey, path }) => {
  const message = useMessage(icdKey, PForm.useFieldValue(path.message));
  useSuit({
    path: path.bindings,
    message,
    suit,
    bound: (bindings) =>
      bindings.flatMap((b) => (b.type === "channel" ? [b.channel] : [])),
    unbind: (bindings, fields, channels) => unbind(bindings, fields, channels, encodes),
  });
  return null;
};
