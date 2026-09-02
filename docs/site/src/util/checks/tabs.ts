// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Check } from "./check.ts";
import { locate } from "./crawl.ts";
import { islands, QUOTED } from "./html.ts";

const NAMED_SLOT = new RegExp(`data-astro-template=${QUOTED}|<astro-slot name=`);
const STYLE_ATTR = new RegExp(`\\sstyle=${QUOTED}`, "g");

// Returns the first margin declaration with a nonzero vertical component, or null.
const verticalMargin = (style: string): string | null => {
  for (const decl of style.split(";")) {
    const [prop, value] = decl.split(":");
    if (value == null) continue;
    const name = prop.trim();
    let vertical: string[] = [];
    if (name === "margin-top" || name === "margin-bottom") vertical = [value];
    else if (name === "margin") {
      const parts = value.trim().split(/\s+/);
      vertical = [parts[0], parts[2] ?? parts[0]];
    }
    // Non-numeric values (auto, var(--x)) parse as NaN and are not spacers.
    const spacer = vertical.some((v) => {
      const n = parseFloat(v);
      return !Number.isNaN(n) && n !== 0;
    });
    if (spacer) return decl.trim();
  }
  return null;
};

// Guards against Astro dropping named MDX slots on tab islands, which ships every
// platform tab blank, and against spacer margins that pad a short panel to match
// its siblings. The zero-islands canary needs a full crawl; filtered runs skip it.
export const tabs = (fullCrawl: boolean): Check => {
  let seen = 0;
  return {
    name: "tabs",
    page: ({ route, html }) => {
      const failures: string[] = [];
      for (const island of islands(html)) {
        if (!island.component.endsWith(".Tabs")) continue;
        seen += 1;
        if (!NAMED_SLOT.test(island.body))
          failures.push(
            `${locate(route, `<${island.component}`)} - ` +
              `${island.component} island has no named slot content`,
          );
        for (const m of island.body.matchAll(STYLE_ATTR)) {
          const decl = verticalMargin(m[1] ?? m[2]);
          if (decl != null)
            failures.push(
              `${locate(route, decl.split(":")[1].trim())} - ` +
                `spacer "${decl}" inside a ${island.component} panel`,
            );
        }
      }
      return failures;
    },
    finish: async (_, report) => {
      if (fullCrawl && seen === 0)
        report("no *.Tabs islands found: markup has changed");
    },
  };
};
