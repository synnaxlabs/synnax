// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "node:path";

import { type Plugin } from "vite";

/**
 * Restores the CSS imports that lib mode strips from emitted modules, so a consumer's
 * bundler collects the stylesheet of every module it imports and nothing more.
 */
export const injectCSS = (): Plugin => ({
  name: "vite-plugin-inject-css",
  enforce: "post",
  generateBundle: (_, bundle) => {
    for (const chunk of Object.values(bundle)) {
      if (chunk.type !== "chunk") continue;
      const sheets = [...(chunk.viteMetadata?.importedCss ?? [])];
      if (sheets.length === 0) continue;
      const dir = path.dirname(chunk.fileName);
      const imports = sheets.map((sheet) => {
        const rel = path.relative(dir, sheet).split(path.sep).join("/");
        return `import "${rel.startsWith(".") ? rel : `./${rel}`}";`;
      });
      chunk.code = `${imports.join("\n")}\n${chunk.code}`;
    }
  },
});
