// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Plugin } from "vite";

const SIDE_EFFECT_IMPORT = /\bimport\s*["']([^"']+)["']/g;
const FIX =
  "Module entries must only re-export. Move each import into the module that uses " +
  "it, or page-wide CSS into the stylesheet an app imports once:";

/**
 * Fails the build when a module entry imports a stylesheet or imports anything only for
 * its side effects. The package marks its JavaScript side-effect free, so a consumer's
 * production bundler skips an entry that only re-exports and drops those imports.
 */
export const checkEntries = (): Plugin => ({
  name: "vite-plugin-check-entries",
  generateBundle: (_, bundle) => {
    const offenses: string[] = [];
    for (const chunk of Object.values(bundle)) {
      if (chunk.type !== "chunk" || !chunk.isEntry) continue;
      const effects = [
        ...(chunk.viteMetadata?.importedCss ?? []),
        ...Array.from(chunk.code.matchAll(SIDE_EFFECT_IMPORT), ([, spec]) => spec),
      ];
      offenses.push(...effects.map((effect) => `${chunk.fileName} imports ${effect}`));
    }
    if (offenses.length === 0) return;
    throw new Error(`${FIX}\n${offenses.join("\n")}`);
  },
});
