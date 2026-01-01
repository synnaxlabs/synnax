// Copyright 2026 Synnax Labs, Inc.
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

import { type NavState } from "@/layout/types/v0";
import * as v1 from "@/layout/types/v1";
import * as v4 from "@/layout/types/v4";

export const VERSION = "5.0.0";

export const sliceStateZ = v4.sliceStateZ.omit({ version: true }).extend({
  version: z.literal(VERSION),
  themes: z.record(z.string(), Theming.themeZ),
});

export type SliceState = z.infer<typeof sliceStateZ>;

const ZERO_NAV_STATE: NavState = {
  main: {
    drawers: {
      left: {
        activeItem: null,
        menuItems: [
          "ontology",
          "channel",
          "range",
          "workspace",
          "device",
          "task",
          "user",
        ],
      },
      right: { activeItem: null, menuItems: [] },
      bottom: { activeItem: null, menuItems: ["visualization"] },
    },
  },
};

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  ...v4.ZERO_SLICE_STATE,
  version: VERSION,
  nav: ZERO_NAV_STATE,
});

export const sliceMigration: migrate.Migration<v4.SliceState, SliceState> =
  migrate.createMigration({
    name: v1.SLICE_MIGRATION_NAME,
    migrate: (s) => ({
      ...s,
      version: VERSION,
      nav: ZERO_NAV_STATE,
      activeTheme: Theming.SYNNAX_DARK.key,
      themes: {
        [Theming.SYNNAX_DARK.key]: Theming.themeZ.parse(
          Theming.SYNNAX_THEMES.synnaxDark,
        ),
        [Theming.SYNNAX_LIGHT.key]: Theming.themeZ.parse(
          Theming.SYNNAX_THEMES.synnaxLight,
        ),
      },
    }),
  });
