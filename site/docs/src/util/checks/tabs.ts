// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Check } from "@/util/checks/check";
import { locate } from "@/util/checks/crawl";
import { attrOf, elements, QUOTED } from "@/util/checks/html";

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

// Names the tab that controls a panel, to find its slot in the page source.
const tabKey = (html: string, panel: string): string | undefined => {
  const id = attrOf(panel, "id");
  const tab = new RegExp(`<[^>]*\\saria-controls="${id}"[^>]*>`).exec(html);
  return tab == null ? undefined : attrOf(tab[0], "data-tab-key");
};

// Guards against Astro dropping named MDX slots, which ships a tab panel blank, and
// against spacer margins that pad a short panel to match its siblings. The zero-panels
// canary needs a full crawl; filtered runs skip it.
export const tabs = (fullCrawl: boolean): Check => {
  let seen = 0;
  return {
    name: "tabs",
    page: ({ route, html }) => {
      const failures: string[] = [];
      for (const panel of elements(html, "div", "role", "tabpanel")) {
        seen += 1;
        if (panel.body.trim() === "") {
          const key = tabKey(html, panel.tag);
          failures.push(
            `${locate(route, `slot="${key}"`)} - tab panel ${key ?? ""} is empty`,
          );
        }
        for (const m of panel.body.matchAll(STYLE_ATTR)) {
          const decl = verticalMargin(m[1] ?? m[2]);
          if (decl != null)
            failures.push(
              `${locate(route, decl.split(":")[1].trim())} - ` +
                `spacer "${decl}" inside a tab panel`,
            );
        }
      }
      return failures;
    },
    finish: async (_, report) => {
      if (fullCrawl && seen === 0) report("no tab panels found: markup has changed");
    },
  };
};
