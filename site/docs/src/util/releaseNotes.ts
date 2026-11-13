// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The indexer runs under plain Node, so this module imports nothing from the site.
import fs from "fs";
import path from "path";

/** The directory that holds each version's notes. */
export const RELEASE_NOTES_DIR = path.join(import.meta.dirname, "../releases");

export interface Notes {
  /** The file name without its extension, which was the notes' route before. */
  name: string;
  /** The id of the notes on /releases. */
  anchor: string;
}

/** Returns the id that the notes of a version render under on /releases. */
export const releaseAnchor = (version: string): string => version.split(".").join("-");

const VERSION = /version="([^"]+)"/;

/**
 * Lists the notes in a directory.
 * @throws {Error} if a file does not pass a version to its release.
 */
export const listReleaseNotes = (dir: string = RELEASE_NOTES_DIR): Notes[] =>
  fs
    .readdirSync(dir)
    .filter((f) => f.endsWith(".mdx"))
    .map((f) => {
      const version = VERSION.exec(fs.readFileSync(path.join(dir, f), "utf8"))?.[1];
      if (version == null) throw new Error(`${f} passes no version to its release`);
      return { name: f.replace(/\.mdx$/, ""), anchor: releaseAnchor(version) };
    });

const DATE = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  year: "numeric",
  timeZone: "America/Denver",
});

/** Formats the day a release was published in Mountain Time, such as `Nov 11, 2026`. */
export const releaseDate = (publishedAt: Date): string => DATE.format(publishedAt);

export interface Release {
  version: string;
  /**
   * When the release shipped. A release without a date has not shipped. Notes from
   * v0.59 pass none: their date is the publish date of the stable Core release.
   */
  date?: string;
  desktopVersion?: string;
  /** The TypeScript client's version, or null if none shipped. Defaults to version. */
  npmVersion?: string | null;
}

export interface Links {
  github: string;
  python: string;
  typescript?: string;
  desktop?: string;
}

/**
 * Returns the links from a release to its Git tags and packages, or null if the release
 * has not shipped, since its tags and packages do not exist yet.
 */
export const releaseLinks = ({
  version,
  date,
  desktopVersion,
  npmVersion = version,
}: Release): Links | null => {
  if (date == null) return null;
  const [major, minor] = version.split(".").map(Number);
  // From v0.59, each product has its own tag, and the Core's tag names the release.
  const tag = major === 0 && minor < 59 ? `synnax-v${version}` : `core/v${version}`;
  const majorMinor = version.split(".").slice(0, 2).join(".");
  return {
    github: `https://github.com/synnaxlabs/synnax/tree/${tag}`,
    python: `https://pypi.org/project/synnax/${majorMinor}/`,
    typescript:
      npmVersion == null
        ? undefined
        : `https://npmjs.com/package/@synnaxlabs/client/v/${npmVersion}`,
    desktop:
      desktopVersion == null
        ? undefined
        : `https://github.com/synnaxlabs/synnax/releases/tag/desktop/v${desktopVersion}`,
  };
};
