// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const prettyParse = <Z extends z.ZodTypeAny>(
  schema: Z,
  value: unknown,
  prefix: string = "",
): z.infer<Z> => {
  try {
    return schema.parse(value);
  } catch (e) {
    if (e instanceof z.ZodError) {
      const errors = e.errors.map((err) => {
        if (err.path.length === 0) return err.message;
        return `${err.path.join(".")}: ${err.message}`;
      });
      throw new Error(`${prefix} - ${errors.join("\n")}`);
    } else throw e;
  }
};
