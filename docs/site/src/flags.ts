// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FLAG_PLUTO, VERCEL_ENV } from "astro:env/client";

// Preview deploys show every flagged surface, so reviewers see dark work.
const PREVIEW = VERCEL_ENV === "preview";

/**
 * Static build-time flags that hide unfinished docs in production. Each entry names its
 * owner and what removes it. `FLAG_<NAME>=true` at build time turns one on; set it in
 * Vercel for the site and as a repository variable for the search index job.
 */
export const FLAGS = {
  // Owner: Patrick Dotson. Removed when the Pluto section returns to the nav.
  pluto: FLAG_PLUTO || PREVIEW,
} satisfies Record<string, boolean>;

export type Flag = keyof typeof FLAGS;

const isFlag = (name: string): name is Flag => Object.hasOwn(FLAGS, name);

/**
 * Reports whether a flag is on.
 * @throws {Error} if the flag is not registered, since a typo must not hide a page.
 */
export const enabled = (flag: string): boolean => {
  if (!isFlag(flag)) throw new Error(`unknown flag: ${flag}`);
  return FLAGS[flag];
};
