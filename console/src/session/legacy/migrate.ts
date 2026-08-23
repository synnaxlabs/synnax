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
 * The connection parameters of a stored Core. The cluster slice flattened them onto
 * the record in 2.0.0; before that they sat under props.
 */
const paramsZ = z.object({
  host: z.string(),
  port: z.union([z.number(), z.string()]),
  username: z.string().default(""),
  password: z.string().default(""),
  secure: z.boolean().default(false),
});

const coreZ = z.union([
  z.object({ name: z.string(), ...paramsZ.shape }),
  z
    .object({ name: z.string(), props: paramsZ })
    .transform(({ name, props }) => ({ name, ...props })),
]);

/** The branches of the 0.56 blob this release still has a home for. */
const rootZ = z.record(z.string(), z.unknown());
const clusterZ = z.object({
  activeCluster: z.string().nullish(),
  clusters: z.record(z.string(), z.unknown()),
});
const themeZ = z.object({ activeTheme: z.string().optional() });
const colorZ = z.object({ colorContext: PColor.contextStateZ.optional() });
// The workspace slice held the active key itself until 1.0.0, and the workspace after.
const workspaceZ = z.object({
  active: z
    .union([z.string(), z.object({ key: z.string() }).transform(({ key }) => key)])
    .nullish(),
});

/**
 * Parses one branch of the blob. Every release line since 0.4 wrote a different shape,
 * so a branch this one cannot read costs only itself.
 */
const parseBranch = <T>(
  raw: unknown,
  schema: z.ZodType<T>,
  name: string,
): T | undefined => {
  if (raw == null) return undefined;
  const parsed = schema.safeParse(raw);
  if (parsed.success) return parsed.data;
  console.error(`could not carry over the previous release's ${name}`, parsed.error);
  return undefined;
};

/**
 * The slices carried over from the previous release. Layouts, mosaics, and per-document
 * view state are absent by design: the Core migrates workspaces into projects and
 * panels, so their content arrives from there rather than from local state.
 */
export interface Seed extends Partial<
  Core.StoreState & Theme.StoreState & Color.StoreState & Project.StoreState
> {}

const themeMode = (activeTheme?: string): Theme.Mode | undefined => {
  const theme = activeTheme?.toLowerCase();
  if (theme == null) return undefined;
  if (theme.includes("dark")) return "dark";
  if (theme.includes("light")) return "light";
  return undefined;
};

const cores = (legacy?: z.infer<typeof clusterZ>): Core.SliceState | undefined => {
  if (legacy == null) return undefined;
  const out: Core.SliceState = { ...Core.ZERO_SLICE_STATE, cores: {} };
  // The old keys were generated too, so they carry over as they are.
  Object.entries(legacy.clusters).forEach(([key, raw]) => {
    const core = parseBranch(raw, coreZ, `Core ${key}`);
    if (core == null) return;
    const { port, ...rest } = core;
    out.cores[key] = { ...rest, key, port: Number(port) };
  });
  if (Object.keys(out.cores).length === 0) return undefined;
  const { activeCluster } = legacy;
  if (activeCluster != null && out.cores[activeCluster] != null)
    out.selected = activeCluster;
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
  const root = parseBranch(await readState(read), rootZ, "session state");
  if (root == null) return {};
  const out: Seed = {};
  const core = cores(parseBranch(root.cluster, clusterZ, "Cores"));
  if (core != null) out[Core.SLICE_NAME] = core;
  const mode = themeMode(parseBranch(root.layout, themeZ, "theme")?.activeTheme);
  if (mode != null) out[Theme.SLICE_NAME] = { ...Theme.ZERO_SLICE_STATE, mode };
  const context = parseBranch(root.layout, colorZ, "colors")?.colorContext;
  if (context != null) out[Color.SLICE_NAME] = { ...Color.ZERO_SLICE_STATE, context };
  // The Core migrates workspaces into projects under the same key, so the selection
  // still names something once a connection is up.
  const selected = parseBranch(root.workspace, workspaceZ, "project selection")?.active;
  if (selected != null)
    out[Project.SLICE_NAME] = { ...Project.ZERO_SLICE_STATE, selected };
  return out;
};
