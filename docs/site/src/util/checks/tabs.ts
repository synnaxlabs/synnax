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
import { islands } from "./html.ts";

const NAMED_SLOT = /data-astro-template="[^"]+"|<astro-slot name="/;

// Guards against Astro dropping named MDX slots on tab islands, which ships every
// platform tab blank. The zero-islands canary only means something when the whole
// site was crawled, so a route-filtered run skips it.
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
      }
      return failures;
    },
    finish: async (_, report) => {
      if (fullCrawl && seen === 0)
        report("no *.Tabs islands found: markup has changed");
    },
  };
};
