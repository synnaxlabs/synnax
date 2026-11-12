// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { json, payload } from "@synnaxlabs/x";

// Re-encoding rejects what a strict decoder rejects: missing padding, stray characters,
// and nonzero trailing bits.
const isBase64 = (text: string): boolean => {
  try {
    return btoa(atob(text)) === text;
  } catch {
    return false;
  }
};

const LONE_SURROGATE =
  /[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/;

// The Core and the Driver reject a lone surrogate and a number past float64 range,
// which JSON.parse accepts.
const isJSON = (text: string): boolean => {
  let wellFormed = true;
  try {
    JSON.parse(text, (key, value: unknown) => {
      if (
        LONE_SURROGATE.test(key) ||
        (typeof value === "string" && LONE_SURROGATE.test(value)) ||
        (typeof value === "number" && !Number.isFinite(value))
      )
        wellFormed = false;
      return value;
    });
  } catch {
    return false;
  }
  return wellFormed;
};

const valueIssue = (field: payload.WriteField): string | null => {
  switch (field.type) {
    case "float":
      return Number.isFinite(field.value) ? null : "Value must be finite";
    case "json":
      return isJSON(field.value) ? null : "Value is not JSON";
    case "bytes":
      return isBase64(field.value) ? null : "Value is not base64";
    default:
      return null;
  }
};

/**
 * Validates a write field of a task to deploy: its pointer is set, and its value can go
 * into a payload.
 */
export const deployWriteFieldZ = payload.writeFieldZ.check((ctx) => {
  const { pointer } = ctx.value;
  const pointerIssue =
    pointer === ""
      ? "Pointer cannot be empty"
      : json.pointerZ.safeParse(pointer).error?.issues[0].message;
  if (pointerIssue != null)
    ctx.issues.push({
      code: "custom",
      input: ctx.value,
      message: pointerIssue,
      path: ["pointer"],
    });
  const issue = valueIssue(ctx.value);
  if (issue != null)
    ctx.issues.push({
      code: "custom",
      input: ctx.value,
      message: issue,
      path: ["value"],
    });
});
