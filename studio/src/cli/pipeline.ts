// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { copyFile, mkdir, mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { bundle } from "@remotion/bundler";
import { renderMedia, renderStill, selectComposition } from "@remotion/renderer";

import { type CaptureSession } from "@/capture/rig";
import { synthesize } from "@/director/cursor";
import { direct } from "@/director/director";
import { overlay } from "@/director/overlay";
import { stage } from "@/director/stage";
import { type Edit, type Format, type Overlays } from "@/film";
import { type Core } from "@/fixtures/core";
import { type FilmProps } from "@/remotion/Film";
import { type HeroProps, type Region, SHOT, type ShotProps } from "@/remotion/Mail";
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
  /**
   * URL of a Console dev server started with VITE_DESKTOP=true, for scripts that show
   * Synnax Desktop. Defaults to DESKTOP_URL.
   */
  desktopUrl?: string;
  theme: "light" | "dark";
  headed?: boolean;
  /** Accepts self-signed certificates from the capture Core. */
  insecure?: boolean;
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

/** Default URL of the Desktop build of the Console dev server. */
const DESKTOP_URL = "http://localhost:5175";

/**
 * runCapture provisions the core, drives the script through a capture session,
 * and returns the recorded timeline (also written to <outDir>/timeline.json).
 */
export const runCapture = async (opts: CaptureRunOptions): Promise<Timeline> => {
  const port = opts.port ?? (opts.core === "external" ? 9090 : STUDIO_PORT);
  // Fixture defaults (sineTelemetry, createRanges, ...) resolve this so scripts
  // need no port plumbing.
  process.env.SYNNAX_STUDIO_PORT = String(port);
  const mod = (await import(path.resolve(opts.scriptPath))) as {
    default: VideoScript;
    /** Script-local capture size; the --width/--height flags win over it. */
    viewport?: { width: number; height: number };
    /** Runs the Driver inside the capture's Core. */
    driver?: boolean;
    /** The integrations the Driver enables. Every one when left out. */
    integrations?: string[];
    /** Shows the Console as Synnax Desktop. */
    desktop?: boolean;
  };
  const desktop = mod.desktop ?? false;
  const { desktopUrl = DESKTOP_URL } = opts;
  let core: Core | undefined;
  if (opts.core !== "external") {
    const { startCore } = await import("@/fixtures/core");
    core = await startCore({
      outDir: opts.outDir,
      binary: opts.coreBin,
      port,
      driver: mod.driver ?? false,
      integrations: mod.integrations,
    });
  }
  try {
    const { CaptureSession } = await import("@/capture/rig");
    const width = opts.width ?? mod.viewport?.width;
    const height = opts.height ?? mod.viewport?.height;
    const session = await CaptureSession.launch({
      url: desktop ? desktopUrl : opts.url,
      outDir: opts.outDir,
      theme: opts.theme,
      corePort: port,
      desktop,
      headed: opts.headed ?? false,
      insecure: opts.insecure ?? false,
      hideCaret: opts.hideCaret ?? false,
      ...(width != null && { width }),
      ...(height != null && { height }),
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
const parseTarget = (value: string | undefined, native: number): number => {
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
  /**
   * Output width target; defaults to 1080p, which is what the docs site
   * serves. A capture narrower than that renders at its native width.
   */
  target?: string;
  /**
   * Draft renders trade quality for speed for review iterations: a higher crf
   * and a fast encoder preset. Never upload a draft.
   */
  draft?: boolean;
  onProgress?: (progress: number) => void;
}

/**
 * withBundle bundles the Remotion entry, serving captureDir as its public dir, and runs
 * render against the bundle's serve URL. The bundle holds a copy of every captured
 * frame, so it is deleted when render settles.
 */
const withBundle = async (
  captureDir: string,
  render: (serveUrl: string) => Promise<void>,
): Promise<void> => {
  const src = path.resolve(import.meta.dirname, "..");
  const serveUrl = await bundle({
    entryPoint: path.join(src, "remotion/entry.ts"),
    publicDir: captureDir,
    webpackOverride: (config) => ({
      ...config,
      resolve: {
        ...config.resolve,
        alias: { ...(config.resolve?.alias as object), "@": src },
      },
    }),
  });
  try {
    await render(serveUrl);
  } finally {
    await rm(serveUrl, { recursive: true, force: true });
  }
};

const nativeWidth = ({ meta }: Timeline): number => Math.round(meta.width * meta.dsf);

/** outputWidth returns the width a capture's video renders at for a target. */
const outputWidth = (timeline: Timeline, target?: string): number => {
  const native = nativeWidth(timeline);
  return target == null ? Math.min(1920, native) : parseTarget(target, native);
};

/** runRender directs the timeline and renders the video through Remotion. */
export const runRender = async (opts: RenderRunOptions): Promise<void> => {
  const { timeline, captureDir, outputLocation, draft = false } = opts;
  const native = nativeWidth(timeline);
  const targetWidth = outputWidth(timeline, opts.target);
  const scale = targetWidth / native;
  const tracks = direct(timeline, targetWidth);
  const inputProps = { meta: timeline.meta, tracks, events: timeline.events };
  await withBundle(captureDir, async (serveUrl) => {
    const composition = await selectComposition({ serveUrl, id: "studio", inputProps });
    await renderMedia({
      composition,
      serveUrl,
      codec: "h264",
      crf: draft ? 22 : 20,
      x264Preset: draft ? "veryfast" : "slow",
      scale,
      pixelFormat: "yuv420p",
      colorSpace: "bt709",
      imageFormat: "png",
      inputProps,
      outputLocation,
      onProgress: ({ progress }) => opts.onProgress?.(progress),
    });
  });
};

export interface FilmOptions {
  edit: Edit;
  format: Format;
  overlays: Overlays;
  timeline: Timeline;
  /** Capture directory holding frames/ (served as the compositor's publicDir). */
  captureDir: string;
}

export interface FilmRenderOptions extends FilmOptions {
  /** Path the encoded MP4 is written to. */
  outputLocation: string;
  /** Higher crf and a fast encoder preset, for review iterations. */
  draft?: boolean;
  onProgress?: (progress: number) => void;
}

/** filmProps stages the edit against its capture into the film's input props. */
const filmProps = ({ edit, format, timeline, ...opts }: FilmOptions): FilmProps => {
  const plan = stage(edit, timeline, format);
  return {
    meta: timeline.meta,
    plan,
    overlays: overlay(opts.overlays, plan.samples, timeline),
    cursor: synthesize(timeline),
    events: timeline.events,
  };
};

/** runFilmRender stages the edit against its capture and renders the film. */
export const runFilmRender = async (opts: FilmRenderOptions): Promise<void> => {
  const { captureDir, outputLocation, draft = false } = opts;
  const inputProps = filmProps(opts);
  await withBundle(captureDir, async (serveUrl) => {
    const composition = await selectComposition({ serveUrl, id: "film", inputProps });
    await renderMedia({
      composition,
      serveUrl,
      codec: "h264",
      crf: draft ? 22 : 18,
      x264Preset: draft ? "veryfast" : "slow",
      pixelFormat: "yuv420p",
      colorSpace: "bt709",
      imageFormat: "png",
      inputProps,
      outputLocation,
      onProgress: ({ progress }) => opts.onProgress?.(progress),
    });
  });
};

export interface FilmStillsOptions extends FilmOptions {
  /** Film times, in seconds, to render a still of. */
  seconds: number[];
  /** Directory each still is written to, as <seconds>s.png. */
  outDir: string;
}

/**
 * runFilmStills renders single frames of the film as PNGs, for reviewing a change
 * without rendering the whole film. Returns the path of each still. Throws when a time
 * falls outside the film.
 */
export const runFilmStills = async (opts: FilmStillsOptions): Promise<string[]> => {
  const inputProps = filmProps(opts);
  const frames = inputProps.plan.samples.length;
  const paths: string[] = [];
  await withBundle(opts.captureDir, async (serveUrl) => {
    const composition = await selectComposition({ serveUrl, id: "film", inputProps });
    for (const seconds of opts.seconds) {
      const frame = Math.round(seconds * inputProps.plan.fps);
      if (frame < 0 || frame >= frames)
        throw new Error(
          `still at ${seconds}s falls outside the film's ${frames / inputProps.plan.fps}s`,
        );
      const output = path.join(opts.outDir, `${seconds}s.png`);
      await renderStill({ composition, serveUrl, frame, output, inputProps });
      paths.push(output);
    }
  });
  return paths;
};

const MAIL = { imageFormat: "jpeg", jpegQuality: 85, scale: 2 } as const;

export interface MailHero {
  /** source is a still in the bundle's public dir. */
  source: string;
  /** output is where the JPEG goes. */
  output: string;
}

/** runMailHeroes renders each still in `dir` as a window on the hero gradient. */
export const runMailHeroes = async (dir: string, heroes: MailHero[]): Promise<void> =>
  await withBundle(dir, async (serveUrl) => {
    for (const { source, output } of heroes) {
      const inputProps: HeroProps = { source };
      const composition = await selectComposition({
        serveUrl,
        id: "mail-hero",
        inputProps,
      });
      await renderStill({ composition, serveUrl, output, inputProps, ...MAIL });
    }
  });

export interface MailShot {
  /** captureDir holds the frames/ and timeline.json of a capture. */
  captureDir: string;
  /** target is the output width the capture's video renders at. */
  target?: string;
  /** seconds is the time of the video frame to show. */
  seconds: number;
  /**
   * region is the box to show, in pixels of the rendered video. Null shows the full
   * width from the top.
   */
  region: Region | null;
  /** output is where the JPEG goes. */
  output: string;
}

/**
 * runMailShot renders a region of a captured video's frame as a 2x JPEG, from the
 * full-resolution capture. Throws when the time or the region falls outside the video.
 */
export const runMailShot = async (shot: MailShot): Promise<void> => {
  const timeline = await loadTimeline(shot.captureDir);
  const { meta, events } = timeline;
  const frame = Math.round(shot.seconds * meta.fps);
  if (frame < 0 || frame >= meta.frames)
    throw new Error(
      `${shot.seconds}s falls outside the capture's ${meta.frames / meta.fps}s`,
    );
  const width = outputWidth(timeline, shot.target);
  const height = (width * meta.height) / meta.width;
  const { x, y, width: w } = shot.region ?? { x: 0, y: 0, width };
  if (
    w <= 0 ||
    x < 0 ||
    y < 0 ||
    x + w > width ||
    y + (w * SHOT.height) / SHOT.width > height
  )
    throw new Error(
      `region ${x},${y},${w} falls outside the ${width}x${Math.round(height)} video`,
    );
  const k = nativeWidth(timeline) / width;
  const inputProps: ShotProps = {
    meta,
    tracks: direct(timeline, width),
    events,
    region: { x: x * k, y: y * k, width: w * k },
  };
  // Bundling copies the public dir, so serve only the frame the still shows.
  const dir = await mkdtemp(path.join(tmpdir(), "studio-shot-"));
  const name = `${String(frame).padStart(6, "0")}.png`;
  try {
    await mkdir(path.join(dir, "frames"));
    await copyFile(
      path.join(shot.captureDir, "frames", name),
      path.join(dir, "frames", name),
    );
    await withBundle(dir, async (serveUrl) => {
      const composition = await selectComposition({
        serveUrl,
        id: "mail-shot",
        inputProps,
      });
      await renderStill({
        composition,
        serveUrl,
        frame,
        output: shot.output,
        inputProps,
        ...MAIL,
      });
    });
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
};
