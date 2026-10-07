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

import { joinUnits, quantities } from "@/util/quantities";

const html = async (source: string): Promise<string> =>
  (await markdownToHtml(source, { hastPlugins: [quantities] })).html;

describe("joinUnits", () => {
  it("should join a number to its unit", () => {
    expect(joinUnits("Up to 50,000 Hz.")).toBe("Up to 50,000 Hz.");
  });

  it("should join a unit with a symbol", () => {
    expect(joinUnits("at 25 °C and 10 Ω")).toBe("at 25 °C and 10 Ω");
  });

  it("should leave a word that starts with a unit", () => {
    expect(joinUnits("1 second")).toBe("1 second");
  });

  it("should leave a number before a word that is not a unit", () => {
    expect(joinUnits("step 1 and 2 more")).toBe("step 1 and 2 more");
  });
});

describe("quantities", () => {
  it("should join units in prose", async () => {
    expect(await html("Reads at 10 kHz.")).toBe("<p>Reads at 10 kHz.</p>\n");
  });

  it("should leave inline code unchanged", async () => {
    expect(await html("Set `rate: 10 Hz`.")).toContain("<code>rate: 10 Hz</code>");
  });

  it("should leave code blocks unchanged", async () => {
    expect(await html("```\n10 Hz\n```")).toContain("10 Hz\n");
  });
});
