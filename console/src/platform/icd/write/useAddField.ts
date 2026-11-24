// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { useState } from "react";

import { createJSONField } from "@/platform/icd/Fields";
import { formatOf, useFormats, useMessage } from "@/platform/icd/Formats";
import { fieldsPath, jsonFields } from "@/platform/icd/read";

export interface UseAddFieldReturn {
  /** Adds a field to the end of the message, in the form of its ICD. */
  add: () => void;
  /** The field that add added last, whose editor starts open. */
  added?: icd.FieldKey;
}

/**
 * @returns a way to add a field to message, in the form of the ICD that an enclosing
 * WithFormats opens. A Suit of the write binds the new field.
 */
export const useAddField = (
  icdKey: icd.Key,
  message: icd.MessageKey,
): UseAddFieldReturn => {
  const formats = useFormats();
  const fields = jsonFields(useMessage(icdKey, message));
  const [added, setAdded] = useState<icd.FieldKey>();
  const add = () => {
    const field = createJSONField(fields);
    formatOf(formats, icdKey).insert(fieldsPath(message), field, fields.at(-1)?.key);
    setAdded(field.key);
  };
  return { add, added };
};
