// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "fs";
import os from "os";
import path from "path";
import { describe, expect, it } from "vitest";

import { listReleaseNotes, releaseAnchor } from "@/util/releaseNotes";

const write = (files: Record<string, string>): string => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "release-notes-"));
  for (const [name, text] of Object.entries(files))
    fs.writeFileSync(path.join(dir, name), text);
  return dir;
};

describe("releaseAnchor", () => {
  it("should join the version's parts with dashes", () => {
    expect(releaseAnchor("0.59.0")).toBe("0-59-0");
  });
});

describe("listReleaseNotes", () => {
  it("should anchor each file to the version it renders", () => {
    const dir = write({
      "0-31-0.mdx": '<Release version="0.31.1" date="May 1, 2024">',
      "0-59-0.mdx": '<Releases.Release\n    version="0.59.0"\n>',
      "notes.txt": "",
    });
    expect(listReleaseNotes(dir)).toEqual([
      { name: "0-31-0", anchor: "0-31-1" },
      { name: "0-59-0", anchor: "0-59-0" },
    ]);
  });

  it("should throw when a file passes no version", () => {
    const dir = write({ "0-60-0.mdx": "<Release />" });
    expect(() => listReleaseNotes(dir)).toThrow(
      "0-60-0.mdx passes no version to its release",
    );
  });

  it("should give every shipped release an anchor on /releases", () => {
    const notes = listReleaseNotes();
    expect(notes.length).toBeGreaterThan(0);
    expect(notes.find(({ name }) => name === "0-31-0")?.anchor).toBe("0-31-1");
  });
});
