// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync } from "node:fs";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { parseArgs } from "node:util";

import { filter, type Manifest, videoName } from "@/manifest";

const usage = `usage: pnpm gallery [filter]
Writes out/gallery.html: a review page showing the light/dark pair of every
produced manifest entry whose id contains [filter]. Open it in a browser.`;

const ROOT = path.resolve(import.meta.dirname, "../..");
const OUT_ROOT = path.join(ROOT, "out");

const esc = (s: string): string =>
  s.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");

const main = async (): Promise<void> => {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: { help: { type: "boolean", default: false } },
  });
  if (values.help) {
    console.log(usage);
    return;
  }
  const manifest = (await import(path.join(ROOT, "videos.ts"))) as {
    default: Manifest;
  };
  const entries = filter(manifest.default, positionals[0]);

  const sections: string[] = [];
  let shown = 0;
  for (const entry of entries) {
    const light = videoName(entry.id, "light");
    const dark = videoName(entry.id, "dark");
    if (!existsSync(path.join(OUT_ROOT, light))) continue;
    shown++;
    let badge = "";
    const stampFile = path.join(OUT_ROOT, entry.id, "produce.json");
    if (existsSync(stampFile)) {
      const stamp = JSON.parse(await readFile(stampFile, "utf8")) as {
        draft?: boolean;
        producedAt?: string;
      };
      const when = stamp.producedAt?.slice(0, 16).replace("T", " ") ?? "";
      badge = `${stamp.draft === true ? '<span class="draft">DRAFT</span> ' : ""}${when}`;
    }
    sections.push(`<section>
  <h2>${esc(entry.id)} <small>${badge}</small></h2>
  <div class="pair">
    <figure><video src="${esc(light)}" controls muted loop></video>
      <figcaption>light</figcaption></figure>
    <figure><video src="${esc(dark)}" controls muted loop></video>
      <figcaption>dark</figcaption></figure>
  </div>
</section>`);
  }

  const html = `<!doctype html>
<meta charset="utf-8">
<title>studio gallery</title>
<style>
  body { font: 14px system-ui; margin: 2rem; background: #16161c; color: #ddd; }
  h2 { font-weight: 500; } h2 small { color: #888; font-size: 0.7em; }
  .draft { color: #ffb454; border: 1px solid #ffb454; padding: 0 4px; }
  .pair { display: flex; gap: 1rem; }
  figure { flex: 1; margin: 0; min-width: 0; }
  video { width: 100%; border: 1px solid #333; }
  figcaption { color: #888; padding-top: 4px; }
</style>
<h1>studio gallery</h1>
<p>${shown} of ${entries.length} matching entries produced</p>
${sections.join("\n")}`;

  const out = path.join(OUT_ROOT, "gallery.html");
  await writeFile(out, html);
  console.log(`wrote ${out} (${shown} entries)`);
};

main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});
