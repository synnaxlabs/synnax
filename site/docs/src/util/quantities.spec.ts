// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { markdownToHtml } from "satteri";
import { describe, expect, it } from "vitest";

import { quantities } from "@/util/quantities";

const NBSP = "\u00a0";

const html = async (source: string): Promise<string> =>
  (await markdownToHtml(source, { hastPlugins: [quantities] })).html;

describe("quantities", () => {
  it("should join a number to its unit", async () => {
    expect(await html("Up to 50,000 Hz.")).toBe(`<p>Up to 50,000${NBSP}Hz.</p>\n`);
  });

  it("should join a unit with a symbol", async () => {
    expect(await html("At 25 °C and 10 Ω.")).toBe(
      `<p>At 25${NBSP}°C and 10${NBSP}Ω.</p>\n`,
    );
  });

  it("should leave a word that starts with a unit", async () => {
    expect(await html("Wait 1 second.")).toBe("<p>Wait 1 second.</p>\n");
  });

  it("should leave a number before a word that is not a unit", async () => {
    expect(await html("Step 1 and 2 more.")).toBe("<p>Step 1 and 2 more.</p>\n");
  });

  it("should leave inline code unchanged", async () => {
    expect(await html("Set `rate: 10 Hz`.")).toContain("<code>rate: 10 Hz</code>");
  });

  it("should leave code blocks unchanged", async () => {
    expect(await html("```\n10 Hz\n```")).toContain("10 Hz\n");
  });
});
