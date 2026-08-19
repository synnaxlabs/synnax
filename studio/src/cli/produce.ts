// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "node:path";
import { parseArgs } from "node:util";

import { loadTimeline, runCapture, runRender } from "@/cli/pipeline";
import { type Timeline } from "@/timeline";

const usage = `usage: pnpm produce --script <path> --out <dir> [options]
  --script <path>   video script module (default export: async (session) => {})
  --out <dir>       output directory for frames, timeline, and video
  --url <url>       Console URL (default http://localhost:5173)
  --core <mode>     ephemeral | external (default ephemeral): ephemeral starts a
                    fresh in-memory core on 9090 for the capture; external uses
                    whatever is already listening there
  --core-bin <path> synnax binary for --core ephemeral (default: $SYNNAX_CORE_BIN
                    or core/synnax found walking up from the studio)
  --port <n>        core port (default 9095 ephemeral, 9090 external); the dev
                    Console is pointed at it via its dev-connection override
  --theme <t>       light | dark (default light)
  --width <px>      capture viewport width in CSS px (default 1512)
  --height <px>     capture viewport height in CSS px (default 945)
  --dsf <n>         capture device scale factor (default 2)
  --target <t>      output width: 1080p | 1440p | 4k | <pixels>
                    (default native capture resolution, width*dsf)
  --draft           fast review render: 1080p cap + fast encoder preset
  --hide-caret      hide the text caret during capture
  --headed          run the capture browser headed
  --skip-capture    reuse <out>/timeline.json and frames from a prior run
  --capture-only    stop after capture (no render)`;

const parseDim = (flag: string, value: string): number => {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) throw new Error(`invalid ${flag}: ${value}`);
  return n;
};

const main = async (): Promise<void> => {
  const { values } = parseArgs({
    options: {
      script: { type: "string" },
      out: { type: "string" },
      url: { type: "string", default: "http://localhost:5173" },
      core: { type: "string", default: "ephemeral" },
      "core-bin": { type: "string" },
      port: { type: "string" },
      theme: { type: "string", default: "light" },
      width: { type: "string" },
      height: { type: "string" },
      dsf: { type: "string" },
      target: { type: "string" },
      draft: { type: "boolean", default: false },
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
    timeline = await loadTimeline(outDir);
  } else {
    console.log("capturing...");
    timeline = await runCapture({
      scriptPath: values.script,
      outDir,
      url: values.url,
      theme,
      headed: values.headed,
      hideCaret: values["hide-caret"],
      core: values.core === "external" ? "external" : "ephemeral",
      coreBin: values["core-bin"],
      ...(values.port != null && { port: parseDim("--port", values.port) }),
      ...(values.width != null && { width: parseDim("--width", values.width) }),
      ...(values.height != null && { height: parseDim("--height", values.height) }),
      ...(values.dsf != null && { dsf: parseDim("--dsf", values.dsf) }),
    });
    console.log(`captured ${timeline.meta.frames} frames`);
  }

  if (values["capture-only"]) return;

  const outputLocation = path.join(outDir, `video-${theme}.mp4`);
  console.log(`rendering ${timeline.meta.frames} frames...`);
  await runRender({
    timeline,
    captureDir: outDir,
    outputLocation,
    target: values.target,
    draft: values.draft,
    onProgress: (progress) => {
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
