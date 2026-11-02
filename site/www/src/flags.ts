// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FLAG_RIGETTI, VERCEL_ENV } from "astro:env/client";

// Preview deploys show every flagged surface, so reviewers see dark work.
const PREVIEW = VERCEL_ENV === "preview";

/**
 * Turns a `FLAG_<NAME>` value from `astro:env/client` into a registry entry. Dev
 * servers and preview deploys turn every flag on.
 */
const flag = (value: boolean): boolean => value || PREVIEW || import.meta.env.DEV;

/**
 * Static build-time flags that hide unfinished pages in production. Each entry names
 * its owner and what removes it. `FLAG_<NAME>=true` at build time turns one on.
 */
const FLAGS = {
  // Emiliano Bonilla. Hides the Rigetti banner until Rigetti approves it, which removes
  // the flag.
  rigetti: flag(FLAG_RIGETTI),
} satisfies Record<string, boolean>;

type Flag = keyof typeof FLAGS;

const isFlag = (name: string): name is Flag => Object.hasOwn(FLAGS, name);

/**
 * Reports whether a flag is on.
 * @throws {Error} if the flag is not registered, since a typo must not hide a page.
 */
export const enabled = (flag: string): boolean => {
  if (!isFlag(flag)) throw new Error(`unknown flag: ${flag}`);
  return FLAGS[flag];
};
