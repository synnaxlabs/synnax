// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Plugin } from "postcss";

export interface Layer {
  name: string;
  /** Matches the paths of the stylesheets that belong to the layer. */
  files: RegExp;
}

/**
 * Wraps each stylesheet in the first layer whose pattern matches its path and leaves
 * the rest unlayered. Layers are given lowest priority first. Each wrapped stylesheet
 * declares the full order, so the order holds no matter which stylesheet loads first.
 */
export const layers = (order: Layer[]): Plugin => {
  const names = order.map(({ name }) => name).join(", ");
  return {
    postcssPlugin: "layers",
    Once: (root, { AtRule }) => {
      const file = root.source?.input.file;
      if (file == null) return;
      const layer = order.find(({ files }) => files.test(file));
      if (layer == null) return;
      const wrapped = new AtRule({ name: "layer", params: layer.name });
      wrapped.append(root.nodes);
      root.removeAll();
      root.append(new AtRule({ name: "layer", params: names }), wrapped);
    },
  };
};
