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

import { listReleaseNotes, releaseAnchor, releaseLinks } from "@/util/releaseNotes";

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

describe("releaseLinks", () => {
  it("should link a shipped release to its tags and packages", () => {
    expect(
      releaseLinks({
        version: "0.60.0",
        date: "Oct 26, 2026",
        desktopVersion: "1.0.0",
      }),
    ).toEqual({
      github: "https://github.com/synnaxlabs/synnax/tree/core/v0.60.0",
      python: "https://pypi.org/project/synnax/0.60/",
      typescript: "https://npmjs.com/package/@synnaxlabs/client/v/0.60.0",
      desktop: "https://github.com/synnaxlabs/synnax/releases/tag/desktop/v1.0.0",
    });
  });

  it("should link a release before v0.59 to its single tag", () => {
    expect(releaseLinks({ version: "0.58.2", date: "Sep 1, 2026" })?.github).toBe(
      "https://github.com/synnaxlabs/synnax/tree/synnax-v0.58.2",
    );
  });

  it("should omit the TypeScript link when no client shipped", () => {
    expect(
      releaseLinks({ version: "0.51.0", date: "Jan 1, 2026", npmVersion: null })
        ?.typescript,
    ).toBeUndefined();
  });

  it("should not link a release without a date", () => {
    expect(releaseLinks({ version: "0.61.0", desktopVersion: "1.1.0" })).toBeNull();
  });
});
