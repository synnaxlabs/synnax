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
import { parseArgs } from "node:util";

import { bundle } from "@remotion/bundler";
import { renderMedia, selectComposition } from "@remotion/renderer";

import { type CaptureSession } from "@/capture/rig";
import { direct } from "@/director/director";
import { parse, type Timeline } from "@/timeline";

export interface VideoScript {
  (session: CaptureSession): Promise<void>;
}

const usage = `usage: pnpm produce --script <path> --out <dir> [options]
  --script <path>   video script module (default export: async (session) => {})
  --out <dir>       output directory for frames, timeline, and video
  --url <url>       Console URL (default http://localhost:5173)
  --theme <t>       light | dark (default light)
  --width <px>      capture viewport width in CSS px (default 1920)
  --height <px>     capture viewport height in CSS px (default 1080)
  --dsf <n>         capture device scale factor (default 2)
  --target <t>      output width: 1080p | 1440p | 4k | <pixels>
                    (default native capture resolution, width*dsf)
  --hide-caret      hide the text caret during capture
  --headed          run the capture browser headed
  --skip-capture    reuse <out>/timeline.json and frames from a prior run
  --capture-only    stop after capture (no render)`;

const parseDim = (flag: string, value: string): number => {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) throw new Error(`invalid ${flag}: ${value}`);
  return n;
};

const TARGET_PRESETS: Record<string, number> = {
  "1080p": 1920,
  "1440p": 2560,
  "4k": 3840,
  "5k": 5120,
};

/** parseTarget resolves --target to an output width in pixels. */
const parseTarget = (value: string | undefined, native: number): number => {
  if (value == null) return native;
  const width = TARGET_PRESETS[value.toLowerCase()] ?? Number(value);
  if (!Number.isFinite(width) || width <= 0)
    throw new Error(`invalid --target: ${value}`);
  return width;
};

const main = async (): Promise<void> => {
  const { values } = parseArgs({
    options: {
      script: { type: "string" },
      out: { type: "string" },
      url: { type: "string", default: "http://localhost:5173" },
      theme: { type: "string", default: "light" },
      width: { type: "string" },
      height: { type: "string" },
      dsf: { type: "string" },
      target: { type: "string" },
      "hide-caret": { type: "boolean", default: false },
      headed: { type: "boolean", default: false },
      "skip-capture": { type: "boolean", default: false },
      "capture-only": { type: "boolean", default: false },
    },
  });
  if (values.script == null || values.out == null) {
    console.error(usage);
    process.exit(1);
  }
  const outDir = path.resolve(values.out);
  const theme = values.theme === "dark" ? "dark" : "light";

  let timeline: Timeline;
  if (values["skip-capture"]) {
    console.log("reusing existing capture");
    timeline = parse(
      JSON.parse(await readFile(path.join(outDir, "timeline.json"), "utf8")),
    );
  } else {
    console.log("capturing...");
    const { CaptureSession } = await import("@/capture/rig");
    const mod = (await import(path.resolve(values.script))) as {
      default: VideoScript;
    };
    const session = await CaptureSession.launch({
      url: values.url,
      outDir,
      theme,
      headed: values.headed,
      hideCaret: values["hide-caret"],
      ...(values.width != null && { width: parseDim("--width", values.width) }),
      ...(values.height != null && { height: parseDim("--height", values.height) }),
      ...(values.dsf != null && { dsf: parseDim("--dsf", values.dsf) }),
    });
    try {
      await mod.default(session);
    } finally {
      timeline = await session.finish();
    }
    console.log(`captured ${timeline.meta.frames} frames`);
  }

  if (values["capture-only"]) return;

  console.log("directing...");
  const tracks = direct(timeline);

  console.log("bundling compositor...");
  const entry = path.resolve(import.meta.dirname, "../remotion/entry.ts");
  const src = path.resolve(import.meta.dirname, "..");
  const serveUrl = await bundle({
    entryPoint: entry,
    publicDir: outDir,
    webpackOverride: (config) => ({
      ...config,
      resolve: {
        ...config.resolve,
        alias: { ...(config.resolve?.alias as object), "@": src },
      },
    }),
  });

  const inputProps = { meta: timeline.meta, tracks, events: timeline.events };
  const composition = await selectComposition({
    serveUrl,
    id: "studio",
    inputProps,
  });

  const native = Math.round(timeline.meta.width * timeline.meta.dsf);
  const targetWidth = parseTarget(values.target, native);
  // Remotion's scale renders the composition at a multiple of its native size;
  // <1 downscales (supersampled output), >1 upscales.
  const scale = targetWidth / native;

  const outputLocation = path.join(outDir, `video-${theme}.mp4`);
  const outH = Math.round((timeline.meta.height / timeline.meta.width) * targetWidth);
  console.log(
    `rendering ${composition.durationInFrames} frames at ${targetWidth}x${outH}...`,
  );
  await renderMedia({
    composition,
    serveUrl,
    codec: "h264",
    crf: 14,
    x264Preset: "slow",
    scale,
    pixelFormat: "yuv420p",
    colorSpace: "bt709",
    imageFormat: "png",
    inputProps,
    outputLocation,
    onProgress: ({ progress }) => {
      if (Math.round(progress * 100) % 10 === 0)
        process.stdout.write(`\r${Math.round(progress * 100)}%`);
    },
  });
  console.log(`\nwrote ${outputLocation}`);
};

main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});
