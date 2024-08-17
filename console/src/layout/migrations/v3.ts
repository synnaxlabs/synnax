// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { z } from "zod";

import * as v0 from "@/layout/migrations/v0";

export const sliceStateZ = v0.sliceStateZ
  .omit({
    version: true,
  })
  .extend({
    version: z.literal("3.0.0"),
    altKeyToKey: z.record(z.string(), z.string()),
    keyToAltKey: z.record(z.string(), z.string()),
  });

export type SliceState = Omit<z.infer<typeof sliceStateZ>, "layouts"> & {
  version: "3.0.0";
  layouts: Record<string, v0.State>;
};

export const ZERO_SLICE_STATE: SliceState = {
  ...v0.ZERO_SLICE_STATE,
  version: "3.0.0",
  altKeyToKey: {},
  keyToAltKey: {},
};

export const sliceMigration: migrate.Migration<v0.SliceState, SliceState> =
  migrate.createMigration({
    name: "layout.slice",
    migrate: (s) => ({
      ...s,
      version: "3.0.0",
      themes: {
        synnaxDark: Theming.SYNNAX_THEMES.synnaxDark,
        synnaxLight: Theming.SYNNAX_THEMES.synnaxLight,
      },
      altKeyToKey: {},
      keyToAltKey: {},
    }),
  });
