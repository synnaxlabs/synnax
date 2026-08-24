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

/**
 * The connection parameters of a stored Core. The cluster slice flattened them onto the
 * record in 2.0.0; before that they sat under props.
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
const colorZ = z.object({ colorContext: PColor.contextStateZ.optional() });

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
 * The slices carried over from the previous release. Layouts, mosaics, per-document
 * view state, and the workspace selection are absent by design: the Core migrates
 * workspaces into projects and panels, so their content arrives from there, and
 * reselecting a project is a single step. The theme stays behind too: 0.56 stored
 * only the OS theme at last run, which the system default already follows.
 */
export interface Migrated extends Partial<Core.StoreState & Color.StoreState> {}

const cores = (legacy?: z.infer<typeof clusterZ>): Core.SliceState | undefined => {
  if (legacy == null) return undefined;
  const out: Core.SliceState = { ...Core.ZERO_SLICE_STATE, cores: {} };
  // The old keys were generated too, so they carry over as they are.
  Object.entries(legacy.clusters).forEach(([key, raw]) => {
    const core = parseBranch(raw, coreZ, `Core ${key}`);
    if (core == null) return;
    // The slice schema is what the store parses on every read, so a record it
    // rejects must drop here rather than poison the whole registry later.
    const mapped = parseBranch({ ...core, key }, Core.coreZ, `Core ${key}`);
    if (mapped == null) return;
    out.cores[key] = mapped;
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
export const migrate = async (read: Reader = openReader()): Promise<Migrated> => {
  const root = parseBranch(await readState(read), rootZ, "session state");
  if (root == null) return {};
  const out: Migrated = {};
  const core = cores(parseBranch(root.cluster, clusterZ, "Cores"));
  if (core != null) out[Core.SLICE_NAME] = core;
  const context = parseBranch(root.layout, colorZ, "colors")?.colorContext;
  if (context != null) out[Color.SLICE_NAME] = { ...Color.ZERO_SLICE_STATE, context };
  return out;
};
