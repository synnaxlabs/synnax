// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v1 from "@/layout/migrations/v1";
import * as v3 from "@/layout/migrations/v3";

const VERSION = "4.0.0";

type ReplaceColorWithHex<T> = T extends Color.Color
  ? string
  : T extends (infer U)[]
    ? ReplaceColorWithHex<U>[]
    : T extends object
      ? { [K in keyof T]: ReplaceColorWithHex<T[K]> }
      : T;

// Utility function to transform colors to hex
const transformColorsToHex = <T>(obj: T): ReplaceColorWithHex<T> => {
  if (obj instanceof Color.Color) return obj.hex as ReplaceColorWithHex<T>;
  if (typeof obj === "object" && obj !== null) {
    const newObj: any = Array.isArray(obj) ? [] : {};
    for (const key in obj)
      if (obj.hasOwnProperty(key)) newObj[key] = transformColorsToHex(obj[key]);

    return newObj as ReplaceColorWithHex<T>;
  }
  return obj as ReplaceColorWithHex<T>;
};

export const sliceStateZ = v3.sliceStateZ
  .omit({ version: true })
  .extend({ version: z.literal(VERSION), colorContext: Color.contextStateZ })
  .transform(transformColorsToHex);

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v3.ZERO_SLICE_STATE,
  version: VERSION,
  colorContext: Color.ZERO_CONTEXT_STATE,
});

export const sliceMigration: migrate.Migration<v3.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: (s) => ({
      ...s,
      version: VERSION,
      colorContext: transformColorsToHex(Color.ZERO_CONTEXT_STATE),
    }),
  });
