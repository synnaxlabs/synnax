// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { fromHit, group, type Hit, type Result } from "@/components/search/result";

const hit = (overrides: Partial<Hit> = {}): Hit => ({
  objectID: "ni/analog-read-task.mdx",
  title: "NI analog read task",
  description: "Learn how to acquire analog data from NI devices with Synnax.",
  href: "/reference/driver/ni/analog-read-task",
  ...overrides,
});

const snippets = (title: string, content: string, matchLevel = "full") => ({
  title: { value: title, matchLevel: "full" },
  content: { value: content, matchLevel },
});

const result = (key: string, section: string): Result => ({
  key,
  title: key,
  excerpt: "",
  href: `/${key}`,
  section,
  path: [],
  ancestors: [],
});

describe("fromHit", () => {
  it("should place a page under its section and the nav groups below it", () => {
    expect(fromHit(hit())).toMatchObject({
      section: "Device drivers",
      path: ["National Instruments"],
      ancestors: ["driver", "ni"],
    });
  });

  it("should leave the path empty for a page directly under its section", () => {
    const r = fromHit(hit({ href: "/reference/driver/get-started" }));
    expect(r.section).toBe("Device drivers");
    expect(r.path).toEqual([]);
  });

  it("should place release notes and blog posts in their own sections", () => {
    expect(fromHit(hit({ href: "/releases/#0-38-0" }))).toMatchObject({
      section: "Releases",
      ancestors: ["releases"],
    });
    expect(fromHit(hit({ href: "/blog/one-billion-rows" }))).toMatchObject({
      section: "Blog",
      ancestors: ["blog"],
    });
  });

  it("should place a page the nav does not list under Reference", () => {
    expect(
      fromHit(hit({ href: "/reference/hardware/ni/analog-read-task" })),
    ).toMatchObject({ section: "Reference", path: [], ancestors: [] });
  });

  it("should show the matched text when the page's text matched", () => {
    const r = fromHit(
      hit({ _snippetResult: snippets("NI analog read task", "… NI <b>analog</b> …") }),
    );
    expect(r.excerpt).toBe("… NI <b>analog</b> …");
  });

  it("should show the description when only the title matched", () => {
    const r = fromHit(
      hit({
        _snippetResult: snippets("NI <b>analog</b>", "This task follows", "none"),
      }),
    );
    expect(r.excerpt).toBe(
      "Learn how to acquire analog data from NI devices with Synnax.",
    );
  });

  it("should show the start of the text when only the title matched and no description exists", () => {
    const r = fromHit(
      hit({
        description: null,
        _snippetResult: snippets("Version <b>0.38</b>", "Fixed a bug …", "none"),
      }),
    );
    expect(r.excerpt).toBe("Fixed a bug …");
  });

  it("should join the marks of a matched phrase into one mark", () => {
    const r = fromHit(
      hit({
        _snippetResult: snippets(
          "NI <b>analog</b> <b>read</b> task",
          "<b>analog</b> <b>read</b> tasks and digital <b>read</b>",
        ),
      }),
    );
    expect(r.title).toBe("NI <b>analog read</b> task");
    expect(r.excerpt).toBe("<b>analog read</b> tasks and digital <b>read</b>");
  });
});

describe("group", () => {
  it("should order sections by their best result and keep each section's order", () => {
    const entries = group([
      result("a", "Hardware"),
      result("b", "Releases"),
      result("c", "Hardware"),
      result("d", "Releases"),
    ]);
    expect(entries.map(({ key, first }) => [key, first])).toEqual([
      ["a", true],
      ["c", false],
      ["b", true],
      ["d", false],
    ]);
  });
});
