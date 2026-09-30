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

import { type Layer, layers } from "./layers.js";

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
