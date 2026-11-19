// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Kind, kindZ } from "@/payload/types.gen";
import { type CrudeDataType, DataType } from "@/telem/telem";

/**
 * @returns the kind of a read field that writes to a channel of the given data type,
 * or undefined when the data type is not a known one.
 */
export const kindOf = (dataType: CrudeDataType): Kind | undefined => {
  const dt = new DataType(dataType);
  if (dt.isFloat) return "float";
  if (dt.isSignedInteger) return "integer";
  if (dt.isUnsignedInteger) return "unsigned";
  const parsed = kindZ.safeParse(dt.toString());
  return parsed.success ? parsed.data : undefined;
};

/** @returns true when the kind is float, integer, or unsigned. */
export const isNumeric = (kind: Kind): kind is "float" | "integer" | "unsigned" =>
  kind === "float" || kind === "integer" || kind === "unsigned";
