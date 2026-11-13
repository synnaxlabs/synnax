// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import postcss from "postcss";
import { describe, expect, it } from "vitest";

import { type Layer, layers, lyraLayers, plutoLayers } from "./index.js";

const ORDER: Layer[] = [
  { name: "lyra", files: /^\/repo\/lyra\// },
  { name: "vendor", files: /\/node_modules\// },
  { name: "pluto", files: /^\/repo\/pluto\// },
];

const wrap = (from?: string): string =>
  postcss([layers(ORDER)]).process(".a { color: red; }", { from }).css;

describe("layers", () => {
  it("should wrap a matching stylesheet in its layer after the full order", () => {
    expect(wrap("/repo/pluto/src/a.css")).toEqual(
      "@layer lyra, vendor, pluto;@layer pluto {.a { color: red; } }",
    );
  });

  it("should put a stylesheet in the first layer that matches it", () => {
    expect(wrap("/repo/lyra/node_modules/pkg/a.css")).toEqual(
      "@layer lyra, vendor, pluto;@layer lyra {.a { color: red; } }",
    );
  });

  it("should leave a stylesheet that matches no layer unlayered", () => {
    expect(wrap("/repo/console/src/a.css")).toEqual(".a { color: red; }");
  });

  it("should leave a stylesheet with no file path unlayered", () => {
    expect(wrap()).toEqual(".a { color: red; }");
  });
});

const layerOf = (order: Layer[], file: string): string | undefined =>
  order.find(({ files }) => files.test(file))?.name;

describe("lyraLayers", () => {
  const order = lyraLayers("/repo");

  it.each([
    "/repo/lyra/src/theming/base.css",
    "/repo/lyra/src/theming/static/theme-dark.css",
    "/repo/lyra/dist/base.css",
    "/repo/lyra/dist/theme.css",
    "/repo/lyra/dist/theme-light.css",
  ])("should put %s in the base layer", (file) => {
    expect(layerOf(order, file)).toEqual("lyra.base");
  });

  it.each(["/repo/lyra/src/button/Button.css", "/repo/lyra/dist/flex/Box.css"])(
    "should put %s in the components layer",
    (file) => {
      expect(layerOf(order, file)).toEqual("lyra.components");
    },
  );

  it("should leave a stylesheet outside Lyra unlayered", () => {
    expect(layerOf(order, "/repo/site/docs/src/styles/base.css")).toBeUndefined();
  });

  it("should match a Windows root", () => {
    expect(layerOf(lyraLayers("C:\\repo"), "C:/repo/lyra/dist/base.css")).toEqual(
      "lyra.base",
    );
  });
});

describe("plutoLayers", () => {
  const order = plutoLayers("/repo");

  it("should order Lyra, third-party CSS, then Pluto", () => {
    expect(order.map(({ name }) => name)).toEqual([
      "lyra.base",
      "lyra.components",
      "vendor",
      "pluto",
    ]);
  });

  it.each([
    ["/repo/node_modules/pkg/a.css", "vendor"],
    ["/repo/pluto/src/mosaic/Mosaic.css", "pluto"],
    ["/repo/lyra/node_modules/pkg/a.css", "lyra.components"],
    ["/repo/console/src/app/App.css", undefined],
  ])("should put %s in %s", (file, layer) => {
    expect(layerOf(order, file)).toEqual(layer);
  });
});
