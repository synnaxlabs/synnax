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

import { OUT_ROOT, ROOT, run } from "@/cli/common";
import { loadTimeline, runCapture, runFilmRender, runFilmStills } from "@/cli/pipeline";
import {
  DSF,
  edit as parseEdit,
  formatZ,
  overlays as parseOverlays,
  thumbnail as parseThumbnail,
} from "@/film";
import { type Timeline } from "@/timeline";

const usage = `usage: pnpm film <id> [options]
  <id>              film file at films/<id>.ts (default export: the capture script,
                    edit: the shot list, thumbnail: the frame written to
                    out/films/<id>-thumbnail.png)
  --url <url>       Console URL (default http://localhost:5173)
  --core-bin <path> synnax binary for the ephemeral core
  --port <n>        core port (default 9095)
  --headed          run the capture browser headed
  --skip-capture    re-plan and re-render from the last capture
  --draft           fast review render: higher crf + fast encoder preset
  --stills <s,...>  render only stills at these film times in seconds, into
                    out/films/<id>-stills/
  --no-render       capture only`;

const main = async (): Promise<void> => {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      url: { type: "string", default: "http://localhost:5173" },
      "core-bin": { type: "string" },
      port: { type: "string" },
      headed: { type: "boolean", default: false },
      "skip-capture": { type: "boolean", default: false },
      draft: { type: "boolean", default: false },
      stills: { type: "string" },
      "no-render": { type: "boolean", default: false },
    },
  });
  const [id] = positionals;
  if (id == null) {
    console.error(usage);
    process.exit(1);
  }
  const scriptPath = path.join(ROOT, "films", `${id}.ts`);
  const mod = (await import(scriptPath)) as {
    edit?: unknown;
    overlays?: unknown;
    format?: unknown;
    thumbnail?: unknown;
  };
  if (mod.edit == null) throw new Error(`${scriptPath} exports no edit`);
  if (mod.thumbnail == null) throw new Error(`${scriptPath} exports no thumbnail`);
  const edit = parseEdit(mod.edit);
  const overlays = parseOverlays(mod.overlays ?? {});
  const format = formatZ.parse(mod.format);
  const thumbnail = parseThumbnail(mod.thumbnail);

  const captureDir = path.join(OUT_ROOT, "films", id);
  let timeline: Timeline;
  if (values["skip-capture"]) timeline = await loadTimeline(captureDir);
  else {
    console.log("capturing...");
    timeline = await runCapture({
      scriptPath,
      outDir: captureDir,
      url: values.url,
      theme: "dark",
      dsf: DSF,
      core: "ephemeral",
      coreBin: values["core-bin"],
      headed: values.headed,
      ...(values.port != null && { port: Number(values.port) }),
    });
    console.log(`captured ${timeline.meta.frames} frames`);
  }

  if (values["no-render"]) return;
  if (values.stills != null) {
    const seconds = values.stills.split(",").map(Number);
    if (seconds.some((s) => !Number.isFinite(s)))
      throw new Error(`--stills takes seconds, got "${values.stills}"`);
    const paths = await runFilmStills({
      edit,
      format,
      overlays,
      timeline,
      captureDir,
      seconds,
      outDir: path.join(OUT_ROOT, "films", `${id}-stills`),
    });
    for (const p of paths) console.log(`wrote ${p}`);
    return;
  }

  const outputLocation = path.join(OUT_ROOT, "films", `${id}.mp4`);
  const thumbnailLocation = path.join(OUT_ROOT, "films", `${id}-thumbnail.png`);
  console.log("rendering...");
  await runFilmRender({
    edit,
    format,
    overlays,
    timeline,
    captureDir,
    outputLocation,
    thumbnail,
    thumbnailLocation,
    draft: values.draft,
    onProgress: (progress) => {
      if (Math.round(progress * 100) % 10 === 0)
        process.stdout.write(`\r${Math.round(progress * 100)}%`);
    },
  });
  console.log(`\nwrote ${outputLocation} and ${thumbnailLocation}`);
};

run(main);
