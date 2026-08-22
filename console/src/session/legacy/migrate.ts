// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color as PColor } from "@synnaxlabs/pluto";
import { z } from "zod";

import { Color } from "@/session/color";
import { Core } from "@/session/core";
import { openReader, type Reader, readState } from "@/session/legacy/read";
import { Project } from "@/session/project";
import { Theme } from "@/session/theme";

/**
 * What the 0.56 blob held that this release still has a home for. Every branch is
 * optional so a blob from any point in that release line parses, and anything that
 * fails validation is left behind rather than failing the whole carry-over.
 */
const stateZ = z.object({
  cluster: z
    .object({
      activeCluster: z.string().nullish(),
      clusters: z.record(
        z.string(),
        z.object({
          name: z.string(),
          host: z.string(),
          port: z.union([z.number(), z.string()]),
          username: z.string().default(""),
          password: z.string().default(""),
          secure: z.boolean().default(false),
        }),
      ),
    })
    .optional(),
  layout: z
    .object({
      activeTheme: z.string().optional(),
      colorContext: PColor.contextStateZ.optional(),
    })
    .partial()
    .optional(),
  workspace: z.object({ active: z.string().nullish() }).optional(),
});

/**
 * The slices carried over from the previous release. Layouts, mosaics, and per-document
 * view state are absent by design: the Core migrates workspaces into projects and
 * panels, so their content arrives from there rather than from local state.
 */
export interface Seed
  extends Partial<
    Core.StoreState & Theme.StoreState & Color.StoreState & Project.StoreState
  > {}

const themeMode = (activeTheme?: string): Theme.Mode | undefined => {
  const theme = activeTheme?.toLowerCase();
  if (theme == null) return undefined;
  if (theme.includes("dark")) return "dark";
  if (theme.includes("light")) return "light";
  return undefined;
};

const cores = (legacy: z.infer<typeof stateZ>["cluster"]): Core.SliceState | undefined => {
  if (legacy == null) return undefined;
  const entries = Object.entries(legacy.clusters);
  if (entries.length === 0) return undefined;
  const out: Core.SliceState = { ...Core.ZERO_SLICE_STATE, cores: {} };
  entries.forEach(([legacyKey, { port, ...rest }]) => {
    const core = Core.keyed({ ...rest, port: Number(port) });
    out.cores[core.key] = core;
    // The old key was generated, so the selection has to be followed across.
    if (legacyKey === legacy.activeCluster) out.selected = core.key;
  });
  return out;
};

/**
 * Reads the state the previous release left on disk and maps it onto the slices this
 * one keeps. The legacy store is only ever read: nothing here writes or clears it, so
 * rolling back to the previous release finds its state intact.
 * @param read - Reader over the legacy store. Defaults to the real one.
 * @returns the slices worth carrying, or nothing when there is no legacy state.
 */
export const migrate = async (read: Reader = openReader()): Promise<Seed> => {
  const raw = await readState(read);
  if (raw == null) return {};
  const parsed = stateZ.safeParse(raw);
  if (!parsed.success) {
    console.error("could not read the previous release's session state", parsed.error);
    return {};
  }
  const { cluster, layout, workspace } = parsed.data;
  const out: Seed = {};
  const core = cores(cluster);
  if (core != null) out[Core.SLICE_NAME] = core;
  const mode = themeMode(layout?.activeTheme);
  if (mode != null) out[Theme.SLICE_NAME] = { ...Theme.ZERO_SLICE_STATE, mode };
  if (layout?.colorContext != null)
    out[Color.SLICE_NAME] = {
      ...Color.ZERO_SLICE_STATE,
      context: layout.colorContext,
    };
  // The Core migrates workspaces into projects under the same key, so the selection
  // still names something once a connection is up.
  if (workspace?.active != null)
    out[Project.SLICE_NAME] = {
      ...Project.ZERO_SLICE_STATE,
      selected: workspace.active,
    };
  return out;
};
