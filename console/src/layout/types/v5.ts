// Copyright 2025 Synnax Labs, Inc.
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

import * as v4 from "@/layout/types/v4";

const VERSION = "5.0.0";

export const sliceStateZ = v4.sliceStateZ
  .omit({ version: true })
  .extend({
    version: z.literal(VERSION),
    colorContext: Color.contextStateZ,
  })
  .transform(Color.transformColorsToHex);

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v4.ZERO_SLICE_STATE,
  version: VERSION,
  colorContext: Color.ZERO_CONTEXT_STATE,
});

export const sliceMigration: migrate.Migration<v4.SliceState, SliceState> =
  migrate.createMigration({
    name: v4.sliceMigration.name,
    migrate: (s) => ({
      ...s,
      version: VERSION,
      colorContext: Color.transformColorsToHex(Color.ZERO_CONTEXT_STATE),
    }),
  });
