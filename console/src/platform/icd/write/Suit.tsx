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
import { deep } from "@synnaxlabs/x";
import { type FC, useEffect } from "react";

import { encodes } from "@/platform/icd/dataType";
import { useMessage } from "@/platform/icd/Formats";
import { jsonFields } from "@/platform/icd/read";
import { Unbind } from "@/platform/icd/Unbind";
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
 */
export const Suit: FC<SuitProps> = ({ icdKey, path }) => {
  const { get, set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const { message: key, bindings } = PForm.useFieldValue(path);
  const message = useMessage(icdKey, key);
  const fields = jsonFields(message);
  useEffect(() => {
    if (isPreview || message?.payload.format !== "json") return;
    // An Unbind effect runs first and can change the bindings after this render.
    const current = get(path.bindings).value;
    const suited = suit(current, fields);
    if (!deep.equal(suited, current)) set(path.bindings, suited);
  }, [isPreview, bindings, message]);
  if (isPreview) return null;
  return bindings.map((b, i) => {
    const field = fields.find(({ key }) => key === b.field);
    if (field == null || b.type !== "channel" || b.channel === 0) return null;
    return (
      <Unbind
        key={b.field}
        path={deep.variant(path.bindings[i], "type", "channel").channel}
        channel={b.channel}
        takes={(dataType) => encodes(field, dataType)}
      />
    );
  });
};
