// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export type Primitive = string | number | boolean | null;

export type PrimitiveTypeName = "string" | "number" | "boolean" | "null";

export const primitiveZ = z.union([z.string(), z.number(), z.boolean(), z.null()]);

export const detectType = (value: Primitive): PrimitiveTypeName => {
  switch (typeof value) {
    case "string":
      return "string";
    case "number":
      return "number";
    case "boolean":
      return "boolean";
    default:
      return "null";
  }
};

export const ZERO_VALUES: Record<PrimitiveTypeName, Primitive> = {
  string: "",
  number: 0,
  boolean: false,
  null: null,
};
