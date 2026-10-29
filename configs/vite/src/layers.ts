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

const under = (root: string, dir: string): string =>
  RegExp.escape(`${root.replaceAll("\\", "/")}/${dir}/`);

/**
 * @returns the layers of Lyra's stylesheets in the repository at root, lowest priority
 * first. Its global stylesheets sit below its components, so a component rule beats a
 * default whatever the specificity.
 */
export const lyraLayers = (root: string): Layer[] => {
  const lyra = under(root, "lyra");
  return [
    {
      name: "lyra.base",
      files: new RegExp(
        `^${lyra}(src|dist)/(theming/)?(static/)?(base|theme(-dark|-light)?)\\.css`,
      ),
    },
    { name: "lyra.components", files: new RegExp(`^${lyra}`) },
  ];
};

/**
 * @returns the layers of an app that renders Pluto in the repository at root, lowest
 * priority first. Lyra styles bare elements, so third-party CSS sits above it, and
 * Pluto overrides both.
 */
export const plutoLayers = (root: string): Layer[] => [
  ...lyraLayers(root),
  { name: "vendor", files: /\/node_modules\// },
  { name: "pluto", files: new RegExp(`^${under(root, "pluto")}`) },
];
