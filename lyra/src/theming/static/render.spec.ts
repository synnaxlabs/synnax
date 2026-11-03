// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "fs";
import { describe, expect, it } from "vitest";

import { render } from "@/theming/static";

// Prettier rewraps long values after generation.
const collapse = (css: string): string => css.replace(/\s+/g, " ").trim();

describe("render", () => {
  it.each(Object.entries(render()))("should match the committed %s", (file, body) => {
    const text = fs.readFileSync(new URL(file, import.meta.url), "utf-8");
    const committed = text.slice(text.indexOf("*/") + "*/".length);
    expect(collapse(committed), "Run pnpm generate-theme in lyra/").toBe(
      collapse(body),
    );
  });
});
