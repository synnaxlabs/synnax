// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { suggest } from "@/util/suggest";

const ROUTES = [
  "/reference/concepts/channels",
  "/reference/concepts/ranges",
  "/reference/client/stream-data",
  "/reference/console/visualize/line-plots",
  "/reference/driver/ni/analog-read-task",
  "/reference/driver/ni/analog-write-task",
];

describe("suggest", () => {
  it("should rank first the route of a page name with a typing error", () => {
    expect(suggest("/reference/concepts/channles", ROUTES)[0]).toBe(
      "/reference/concepts/channels",
    );
  });

  it("should find a page that moved to another section", () => {
    expect(suggest("/reference/console/line-plots", ROUTES)).toEqual([
      "/reference/console/visualize/line-plots",
    ]);
  });

  it("should find a page from a shortened name, best first", () => {
    expect(suggest("/reference/driver/ni/analog-read", ROUTES)).toEqual([
      "/reference/driver/ni/analog-read-task",
      "/reference/driver/ni/analog-write-task",
    ]);
  });

  it("should ignore case and a trailing slash", () => {
    expect(suggest("/Reference/Concepts/Ranges/", ROUTES)[0]).toBe(
      "/reference/concepts/ranges",
    );
  });

  it("should find nothing for an unrelated route", () => {
    expect(suggest("/asdf", ROUTES)).toEqual([]);
  });

  it("should return at most limit routes", () => {
    expect(suggest("/reference/driver/ni/analog-read", ROUTES, 1)).toEqual([
      "/reference/driver/ni/analog-read-task",
    ]);
  });

  it("should put the shorter route first when two routes score the same", () => {
    expect(suggest("/aaaa/foo", ["/aaab/foo", "/aaa/foo"])).toEqual([
      "/aaa/foo",
      "/aaab/foo",
    ]);
  });
});
