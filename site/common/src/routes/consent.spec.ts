// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { GET } from "./consent";

const from = (country?: string): Response =>
  GET({
    request: new Request("https://www.synnaxlabs.com/consent", {
      headers: country == null ? {} : { "x-vercel-ip-country": country },
    }),
  });

describe("consent", () => {
  it("should ask a visitor from Europe", async () => {
    expect(await from("DE").text()).toBe("ask");
  });

  it("should give notice to a visitor from elsewhere", async () => {
    expect(await from("US").text()).toBe("notice");
  });

  it("should ask when the country is missing", async () => {
    expect(await from().text()).toBe("ask");
  });

  it("should let only the visitor's browser keep the answer, for a day", () => {
    expect(from("US").headers.get("Cache-Control")).toBe("private, max-age=86400");
  });
});
