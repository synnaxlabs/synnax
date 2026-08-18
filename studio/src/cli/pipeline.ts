// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFile } from "node:fs/promises";
import path from "node:path";

import { bundle } from "@remotion/bundler";
import { renderMedia, selectComposition } from "@remotion/renderer";

import { type CaptureSession } from "@/capture/rig";
import { direct } from "@/director/director";
import { type Core } from "@/fixtures/core";
import { parse, type Timeline } from "@/timeline";

export interface VideoScript {
  (session: CaptureSession): Promise<void>;
}

export interface CaptureRunOptions {
  /** Video script module path (default export: async (session) => {}). */
  scriptPath: string;
  /** Directory receiving frames/, timeline.json, and core.log. */
  outDir: string;
  /** Console dev server URL. */
  url: string;
  theme: "light" | "dark";
  headed?: boolean;
  hideCaret?: boolean;
  width?: number;
  height?: number;
  dsf?: number;
  /** "ephemeral" starts a fresh in-memory core for the capture. */
  core: "ephemeral" | "external";
  /** synnax binary for the ephemeral core. */
  coreBin?: string;
  /**
   * Port the capture's core listens on (the dev Console is pointed at it via
   * its dev-connection override). Defaults to the studio's own port, away from
   * the shared dev core on 9090.
   */
  port?: number;
}

/** Default port for studio-owned ephemeral cores; 9090 stays free for dev. */
export const STUDIO_PORT = 9095;

/**
 * runCapture provisions the core, drives the script through a capture session,
 * and returns the recorded timeline (also written to <outDir>/timeline.json).
 */
export const runCapture = async (opts: CaptureRunOptions): Promise<Timeline> => {
  const port = opts.port ?? (opts.core === "external" ? 9090 : STUDIO_PORT);
  // Fixture defaults (sineTelemetry, seedRanges, ...) resolve this so scripts
  // need no port plumbing.
  process.env.SYNNAX_STUDIO_PORT = String(port);
  let core: Core | undefined;
  if (opts.core !== "external") {
    const { startCore } = await import("@/fixtures/core");
    core = await startCore({ outDir: opts.outDir, binary: opts.coreBin, port });
  }
  try {
    const { CaptureSession } = await import("@/capture/rig");
    const mod = (await import(path.resolve(opts.scriptPath))) as {
      default: VideoScript;
    };
    const session = await CaptureSession.launch({
      url: opts.url,
      outDir: opts.outDir,
      theme: opts.theme,
      corePort: port,
      headed: opts.headed ?? false,
      hideCaret: opts.hideCaret ?? false,
      ...(opts.width != null && { width: opts.width }),
      ...(opts.height != null && { height: opts.height }),
      ...(opts.dsf != null && { dsf: opts.dsf }),
    });
    let timeline: Timeline;
    try {
      await mod.default(session);
    } finally {
      timeline = await session.finish();
    }
    return timeline;
  } finally {
    await core?.stop();
  }
};

/** loadTimeline parses a previously captured <outDir>/timeline.json. */
export const loadTimeline = async (outDir: string): Promise<Timeline> =>
  parse(JSON.parse(await readFile(path.join(outDir, "timeline.json"), "utf8")));

const TARGET_PRESETS: Record<string, number> = {
  "1080p": 1920,
  "1440p": 2560,
  "4k": 3840,
  "5k": 5120,
};

/** parseTarget resolves a target spec ("1080p" | "4k" | pixels) to a width. */
export const parseTarget = (value: string | undefined, native: number): number => {
  if (value == null) return native;
  const width = TARGET_PRESETS[value.toLowerCase()] ?? Number(value);
  if (!Number.isFinite(width) || width <= 0)
    throw new Error(`invalid target: ${value}`);
  return width;
};

export interface RenderRunOptions {
  timeline: Timeline;
  /** Capture directory holding frames/ (served as the compositor's publicDir). */
  captureDir: string;
  /** Path the encoded MP4 is written to. */
  outputLocation: string;
  /** Output width target; defaults to native capture resolution. */
  target?: string;
  /**
   * Draft renders trade quality for speed for review iterations: capped at
   * 1080p (unless target says otherwise) with a fast encoder preset. Never
   * upload a draft.
   */
  draft?: boolean;
  onProgress?: (progress: number) => void;
}

/** runRender directs the timeline and renders the video through Remotion. */
export const runRender = async (opts: RenderRunOptions): Promise<void> => {
  const { timeline, captureDir, outputLocation, draft = false } = opts;
  const tracks = direct(timeline);

  const entry = path.resolve(import.meta.dirname, "../remotion/entry.ts");
  const src = path.resolve(import.meta.dirname, "..");
  const serveUrl = await bundle({
    entryPoint: entry,
    publicDir: captureDir,
    webpackOverride: (config) => ({
      ...config,
      resolve: {
        ...config.resolve,
        alias: { ...(config.resolve?.alias as object), "@": src },
      },
    }),
  });

  const inputProps = { meta: timeline.meta, tracks, events: timeline.events };
  const composition = await selectComposition({ serveUrl, id: "studio", inputProps });

  const native = Math.round(timeline.meta.width * timeline.meta.dsf);
  const explicit = parseTarget(opts.target, native);
  const targetWidth = draft && opts.target == null ? Math.min(1920, native) : explicit;
  const scale = targetWidth / native;

  await renderMedia({
    composition,
    serveUrl,
    codec: "h264",
    crf: draft ? 22 : 14,
    x264Preset: draft ? "veryfast" : "slow",
    scale,
    pixelFormat: "yuv420p",
    colorSpace: "bt709",
    imageFormat: "png",
    inputProps,
    outputLocation,
    onProgress: ({ progress }) => opts.onProgress?.(progress),
  });
};
