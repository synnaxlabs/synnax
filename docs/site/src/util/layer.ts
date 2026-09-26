// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import postcss, { type Plugin } from "postcss";

/**
 * Wraps every stylesheet whose path matches the pattern in the given cascade layer.
 * Lyra ships its CSS unlayered, one file per component; the site keeps them under
 * its own `@layer` order this way.
 */
export const layer = (pattern: RegExp, name: string): Plugin => ({
  postcssPlugin: "layer",
  Once: (root) => {
    const file = root.source?.input.file;
    if (file == null || !pattern.test(file)) return;
    const wrapped = postcss.atRule({ name: "layer", params: name });
    wrapped.append(root.nodes);
    root.removeAll();
    root.append(wrapped);
  },
});
